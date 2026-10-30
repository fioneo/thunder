package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Position uint8

const (
	Center Position = iota
	TopRight
)

type layoutFunc func(p tview.Primitive, width, proportionWidth, height, proportionHeight int) tview.Primitive

func (p Position) ToFunc() layoutFunc {
	switch p {
	case TopRight:
		return topRight
	default:
		return center
	}
}

var popupCount atomic.Uint32

func nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, popupCount.Add(1))
}

type defaultPopup struct {
	id        string
	primitive tview.Primitive
	focusable bool
	focused   bool
	position  Position
	width     int
	propW     int
	height    int
	propH     int
	blurFunc  func()
	focusFunc func()
}

type timedPopup struct {
	*defaultPopup
	timeout time.Duration
	hide    chan struct{}
	once    sync.Once
}

func NewDefaultPopup(p tview.Primitive, id string, focusable bool, position Position, width, propWidth, height, propHeight int) *defaultPopup {
	return &defaultPopup{
		primitive: p,
		id:        id,
		focusable: focusable,
		position:  position,
		width:     width,
		propW:     propWidth,
		height:    height,
		propH:     propHeight,
	}
}

func NewTimeoutPopup(p tview.Primitive, id string, focusable bool, position Position, timeout time.Duration, width, propWidth, height, propHeight int) *timedPopup {
	return &timedPopup{
		defaultPopup: NewDefaultPopup(p, id, focusable, position, width, propWidth, height, propHeight),
		hide:         make(chan struct{}),
		timeout:      timeout,
	}
}

func (p *defaultPopup) ID() string { return p.id }

func (p *defaultPopup) Primitive() tview.Primitive { return p.primitive }

func (p *defaultPopup) Focusable() bool { return p.focusable }

func (p *defaultPopup) Focused() bool { return p.focused }

func (p *defaultPopup) SetBlurFunc(f func()) { p.blurFunc = f }

func (p *defaultPopup) SetFocusFunc(f func()) { p.focusFunc = f }

func (p *defaultPopup) Show() { thunder.Pages.ShowPage(p.id) }

func (p *defaultPopup) Hide() { thunder.Pages.RemovePage(p.id) }

func (p *defaultPopup) Blur() {
	p.focused = false
	if panel, ok := p.primitive.(Panel); ok {
		thunder.Unfocus(panel)
	}
	if p.blurFunc != nil {
		p.blurFunc()
	}
}

func (p *defaultPopup) Focus() {
	p.focused = true
	if panel, ok := p.primitive.(Panel); ok {
		thunder.Focus(panel)
	} else {
		thunder.App.SetFocus(p.primitive)
	}
	if p.focusFunc != nil {
		p.focusFunc()
	}
}

func (p *timedPopup) Hide() {
	p.defaultPopup.Hide()
	p.Close()
}

func (p *timedPopup) Close() {
	p.once.Do(func() { close(p.hide) })
}

type PopupManager struct {
	popups    []Popup
	prevTimed *timedPopup
}

func NewPopupManager() *PopupManager {
	return &PopupManager{}
}

func (pm *PopupManager) top() Popup {
	if len(pm.popups) == 0 {
		return nil
	}
	return pm.popups[len(pm.popups)-1]
}

func (pm *PopupManager) remove(id string) {
	for i, p := range pm.popups {
		if p.ID() == id {
			pm.popups = append(pm.popups[:i], pm.popups[i+1:]...)
			return
		}
	}
}

func (pm *PopupManager) addPage(d *defaultPopup) {
	layout := d.position.ToFunc()
	thunder.Pages.AddPage(d.id, layout(d.primitive, d.width, d.propW, d.height, d.propH), true, false)
	d.Show()
}

func (pm *PopupManager) Show(p Popup) {
	switch popup := p.(type) {
	case *defaultPopup:
		pm.addPage(popup)
		if popup.focusable {
			if top := pm.top(); top != nil {
				top.Blur()
			}
			pm.popups = append(pm.popups, popup)
		}
		pm.SwitchFocus()

	case *timedPopup:
		if pm.prevTimed != nil {
			pm.prevTimed.Close()
		}
		pm.addPage(popup.defaultPopup)
		pm.prevTimed = popup

		pm.SwitchFocus()

		go func() {
			t := time.NewTimer(popup.timeout)
			defer t.Stop()
			select {
			case <-t.C:
			case <-popup.hide:
			}
			onUI(func() { thunder.Popups.Hide(popup) })
		}()
	}
}

func (pm *PopupManager) Hide(p Popup) {
	p.Hide()
	if tp, ok := p.(*timedPopup); ok && pm.prevTimed == tp {
		pm.prevTimed = nil
	}
	pm.remove(p.ID())
	pm.SwitchFocus()
}

func (pm *PopupManager) SwitchFocus() {
	top := pm.top()
	if top == nil {
		thunder.ResetFocus()
		return
	}
	thunder.UnfocusAll()
	top.Focus()
}

func (pm *PopupManager) Count() int {
	return len(pm.popups)
}

func confirmationPopup(text string, handler func(int, string)) {
	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(thunder.Colors.Background).
		AddButtons([]string{"no", "yes"}).
		SetButtonStyle(tcell.StyleDefault.Foreground(thunder.Colors.Accent).Background(thunder.Colors.Background)).
		SetButtonActivatedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Accent))

	modal.SetBorderColor(thunder.Colors.Accent).
		SetBackgroundColor(thunder.Colors.Background)

	modal.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		}
		return e
	})

	popup := NewDefaultPopup(modal, nextID("confirmation_popup"), true, Center, 0, 0, 0, 0)

	modal.SetDoneFunc(func(idx int, label string) {
		thunder.Popups.Hide(popup)
		handler(idx, label)
	})

	thunder.Popups.Show(popup)
}

func errorPopup(err error) {
	if err == nil {
		return
	}
	lines := strings.Split(err.Error(), "\n")
	timeoutPopup(err.Error(), 3*time.Second, tcell.ColorRed, tcell.ColorRed, 0, len(lines))
}

func infoPopup(title, desc string) {
	text := fmt.Sprintf("%s\n%s", title, desc)
	timeoutPopup(text, 3*time.Second, thunder.Colors.Accent, thunder.Colors.Foreground, 0, len(text))
}

func timeoutPopup(text string, timeout time.Duration, textColor, borderColor tcell.Color, width, height int) {
	textView := tview.NewTextView().
		SetText(text).
		SetTextColor(textColor).SetTextAlign(tview.AlignCenter)

	textView.SetBackgroundColor(thunder.Colors.Background).SetBorder(true).
		SetBorderColor(borderColor).SetBorderPadding(1, 1, 1, 1)

	height += 4 // 2 padding + 2 for borders
	popup := NewTimeoutPopup(textView, nextID("timeout_popup"), false, TopRight, timeout, 0, 0, height, 0)

	thunder.Popups.Show(popup)
}

func inputPopup(title, placeholder, text string, handler func(ok bool, text string)) {
	colors := thunder.Colors
	textStyle := tcell.StyleDefault.Foreground(colors.Song).Background(colors.Background)
	placeholderStyle := tcell.StyleDefault.Foreground(tcell.ColorGrey).Background(colors.Background)

	input := NewInput(placeholder, textStyle, placeholderStyle).SetText(text)

	popup := NewDefaultPopup(input, nextID("input_popup"), true, Center, 0, 1, 5, 0)

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			thunder.Popups.Hide(popup)
			handler(false, "")
		case tcell.KeyEnter:
			value := input.GetText()
			thunder.Popups.Hide(popup)
			handler(true, value)
		}
	})

	input.SetBorder(true).SetBorderColor(colors.Foreground).SetTitle(title).SetBorderPadding(1, 1, 1, 1)

	thunder.Popups.Show(popup)
}

func NewInput(placeholder string, textStyle, placeholderStyle tcell.Style) *tview.InputField {
	return tview.NewInputField().SetPlaceholder(placeholder).SetFieldStyle(textStyle).SetPlaceholderStyle(placeholderStyle)
}

// zero width or height means primitive will be resized automatically
func topRight(p tview.Primitive, width, proportionWidth, height, proportionHeight int) tview.Primitive {
	if width == 0 && proportionWidth == 0 {
		proportionWidth = 15
	}
	if height == 0 && proportionHeight == 0 {
		proportionHeight = 2
	}

	return tview.NewFlex().
		AddItem(nil, 0, 20, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, proportionHeight, false).
			AddItem(nil, 0, 12, false), width, proportionWidth, false).
		AddItem(nil, 0, 1, false)
}

// zero width or height means primitive will be resized automatically
func center(p tview.Primitive, width, proportionWidth, height, proportionHeight int) tview.Primitive {
	if width == 0 && proportionWidth == 0 {
		proportionWidth = 2
	}
	if height == 0 && proportionHeight == 0 {
		proportionHeight = 3
	}
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(p, width, proportionWidth, false).
			AddItem(nil, 0, 1, false), height, proportionHeight, false).
		AddItem(nil, 0, 1, false)
}

func downloadPopup() {
	colors := thunder.Colors
	textStyle := tcell.StyleDefault.Foreground(colors.Song).Background(colors.Background)
	placeholderStyle := tcell.StyleDefault.Foreground(tcell.ColorGrey).Background(colors.Background)

	input := NewInput("Name of the song...", textStyle, placeholderStyle)
	list := tview.NewList()
	title := "Download songs"

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 1, 0, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(list, 0, 1, false)
	frame := tview.NewFrame(flex)

	// Cancels only when popup closed.
	ctx, cancel := context.WithCancel(context.Background())

	type downloadInfo struct {
		url      string
		title    string
		duration int
	}

	type searchResult struct {
		Entries []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Duration *int   `json:"duration"`
		} `json:"entries"`
	}

	var (
		songs       []downloadInfo
		searching   bool
		downloading bool
		popup       *defaultPopup
	)

	const (
		inputPanel int = iota
		listPanel
	)
	focused := inputPanel

	unfocus := func(p int) {
		if p == inputPanel {
			thunder.Unfocus(input)
		} else {
			thunder.Unfocus(list)
		}
	}

	setFocus := func(p int) {
		unfocus(inputPanel)
		unfocus(listPanel)
		focused = p
		if p == inputPanel {
			thunder.Focus(input)
		} else {
			thunder.Focus(list)
		}
	}

	cycle := func() {
		if focused == inputPanel {
			if len(songs) > 0 {
				setFocus(listPanel)
			}
		} else {
			setFocus(inputPanel)
		}
	}

	fmtDur := func(sec int) string {
		h, m, s := sec/3600, sec/60%60, sec%60
		if h > 0 {
			return fmt.Sprintf("%d:%02d:%02d", h, m, s)
		}
		return fmt.Sprintf("%d:%02d", m, s)
	}

	lastLine := func(s string) string {
		s = strings.TrimSpace(s)
		if i := strings.LastIndex(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		return s
	}

	// Search songs
	search := func(query string) {
		searching = true
		frame.SetTitle("Fetching...")

		go func() {
			sctx, scancel := context.WithTimeout(ctx, 5*time.Second)
			defer scancel()

			format := fmt.Sprintf("ytsearch%d:%s", 20, query)
			out, sErr := exec.CommandContext(sctx, "yt-dlp", format, "--flat-playlist", "--dump-single-json").Output()

			var res searchResult
			if sErr != nil {
				sErr = fmt.Errorf("yt-dlp search: %w", sErr)
			} else if jerr := json.Unmarshal(out, &res); jerr != nil {
				sErr = fmt.Errorf("parse search result: %w", jerr)
			}

			// Popup already closed
			if ctx.Err() != nil {
				return
			}

			onUI(func() {
				searching = false
				if ctx.Err() != nil {
					return
				}
				frame.SetTitle(title)

				if sErr != nil {
					err := ErrorMessage{
						err:     sErr,
						errView: fmt.Errorf("Search error.\nSee log file."),
					}
					SendErr(err)
					return
				}

				list.Clear()
				songs = make([]downloadInfo, 0, len(res.Entries))
				for _, entry := range res.Entries {
					info := downloadInfo{
						url:   "https://youtube.com/watch?v=" + entry.ID,
						title: entry.Title,
					}
					if entry.Duration != nil {
						info.duration = *entry.Duration
					}
					songs = append(songs, info)
					list.AddItem(fmt.Sprintf("%s %s", tview.Escape(info.title), fmtDur(info.duration)), "", 0, nil)
				}

				if len(songs) == 0 {
					frame.SetTitle("Nothing found")
					return
				}

				if popup.Focused() {
					setFocus(listPanel)
				} else {
					focused = listPanel
				}
			})
		}()
	}

	download := func(song downloadInfo, dir string) {
		fail := func(err error) {
			SendErr(err)
			onUI(func() {
				downloading = false
				frame.SetTitle(title)
			})
		}

		onUI(func() { frame.SetTitle("Starting download...") })

		cmd := exec.Command(
			"yt-dlp",
			"-x",
			"--no-playlist",
			"--audio-format", "mp3",
			"--audio-quality", "0",
			"--newline",
			"--progress-template",
			"downloading%(progress.downloaded_bytes)s %(progress.total_bytes)s %(progress.total_bytes_estimate)s %(progress.speed)s %(progress.eta)s",
			"-o", filepath.Join(dir, "%(title)s.%(ext)s"),
			song.url,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fail(fmt.Errorf("yt-dlp stdout: %w", err))
			return
		}
		if err := cmd.Start(); err != nil {
			fail(fmt.Errorf("start yt-dlp: %w", err))
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "downloading") {
				continue
			}

			fields := strings.Fields(strings.TrimPrefix(line, "downloading"))
			if len(fields) != 5 {
				continue
			}

			downloaded, _ := strconv.ParseInt(fields[0], 10, 64)
			total, _ := strconv.ParseInt(fields[1], 10, 64)
			if total <= 0 {
				total, _ = strconv.ParseInt(fields[2], 10, 64)
			}
			speed, _ := strconv.ParseFloat(fields[3], 64)
			eta, _ := strconv.Atoi(fields[4])

			var status string
			if total > 0 {
				status = fmt.Sprintf("Downloading %.1f%% | %.2f MB/s | ETA %ds",
					float64(downloaded)/float64(total)*100, speed/1024/1024, eta)
			} else {
				status = "Downloading..."
			}
			onUI(func() { frame.SetTitle(status) })
		}
		scanErr := scanner.Err()
		_, _ = io.Copy(io.Discard, stdout)
		waitErr := cmd.Wait()

		if scanErr != nil {
			fail(fmt.Errorf("read yt-dlp output: %w", scanErr))
			return
		}
		if waitErr != nil {
			fail(fmt.Errorf("yt-dlp: %w: %s", waitErr, lastLine(stderr.String())))
			return
		}

		onUI(func() {
			downloading = false
			frame.SetTitle("Download completed")
			infoPopup("Song saved to", dir)
			thunder.Library.Refresh()
		})
	}

	startDownload := func() {
		idx := list.GetCurrentItem()
		if downloading || idx < 0 || idx >= len(songs) {
			return
		}
		song := songs[idx]
		dir := thunder.Library.GetCurrentPath()

		inputPopup("Save path", dir, dir, func(ok bool, savePath string) {
			savePath = strings.TrimSpace(savePath)
			if !ok || savePath == "" {
				return
			}
			downloading = true
			go download(song, savePath)
		})
	}

	nav := func(e *tcell.EventKey) bool {
		switch e.Key() {
		case tcell.KeyEsc:
			cancel()
			thunder.Popups.Hide(popup)
			return true
		case tcell.KeyTab, tcell.KeyBacktab:
			cycle()
			return true
		}
		return false
	}

	input.SetBorderColor(colors.Accent).SetBackgroundColor(colors.Background)
	input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if nav(e) {
			return nil
		}
		switch e.Key() {
		case tcell.KeyEnter:
			query := strings.TrimSpace(input.GetText())
			if query != "" && !searching {
				search(query)
			}
			return nil
		case tcell.KeyDown:
			if len(songs) > 0 {
				setFocus(listPanel)
			}
			return nil
		}
		return e
	})

	list.ShowSecondaryText(false).SetMainTextStyle(tcell.StyleDefault.Foreground(colors.Song).Background(colors.Background)).
		SetSelectedFocusOnly(true).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(colors.Song)).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetBorder(true).
		SetBorderColor(colors.Foreground)

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if nav(e) {
			return nil
		}
		if e.Key() == tcell.KeyRune {
			switch e.Rune() {
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case ' ':
				startDownload()
				return nil
			}
		}
		return e
	})

	flex.SetBorderPadding(0, 0, 1, 1)
	frame.SetTitle(title).SetTitleColor(colors.Accent).SetBorder(true).SetBorderColor(colors.Accent)

	popup = NewDefaultPopup(frame, nextID("download_popup"), true, Center, 0, 3, 0, 0)

	popup.SetBlurFunc(func() {
		unfocus(inputPanel)
		unfocus(listPanel)
	})
	popup.SetFocusFunc(func() {
		setFocus(focused)
	})

	thunder.Popups.Show(popup)
}
