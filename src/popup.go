package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Position uint8

const (
	Center Position = iota
	TopRight
)

func (p Position) ToFunc() func(tview.Primitive, int, int) tview.Primitive {
	var (
		f func(tview.Primitive, int, int) tview.Primitive
	)
	switch p {
	case Center:
		f = center
	case TopRight:
		f = topRight
	}
	return f
}

var (
	popupCount atomic.Uint32
)

type defaultPopup struct {
	id        string
	primitive tview.Primitive
	focusable bool
	position  Position
	width     int
	height    int
}

type timedPopup struct {
	*defaultPopup
	timeout time.Duration
	hide    chan struct{}
	once    sync.Once
}

func NewDefaultPopup(p tview.Primitive, id string, focusable bool, position Position, width, height int) *defaultPopup {
	popup := &defaultPopup{
		primitive: p,
		id:        id,
		focusable: focusable,
		position:  position,
		width:     width,
		height:    height,
	}

	return popup
}

func NewTimeoutPopup(p tview.Primitive, id string, focusable bool, position Position, timeout time.Duration, width, height int) *timedPopup {
	return &timedPopup{
		defaultPopup: &defaultPopup{
			primitive: p,
			id:        id,
			focusable: focusable,
			position:  position,
			width:     width,
			height:    height,
		},
		hide:    make(chan struct{}),
		timeout: timeout,
	}
}

func (p *defaultPopup) ID() string {
	return p.id
}

func (p *defaultPopup) Show() {
	thunder.Pages.ShowPage(p.id)
}

func (p *defaultPopup) Hide() {
	thunder.Pages.RemovePage(p.id)
}

func (p *defaultPopup) Primitive() tview.Primitive {
	return p.primitive
}

func (p *defaultPopup) Focusable() bool {
	return p.focusable
}

func (p *defaultPopup) Width() int {
	return p.width
}

func (p *defaultPopup) Height() int {
	return p.height
}

func (p *timedPopup) ID() string {
	return p.id
}

func (p *timedPopup) Show() {
	thunder.Pages.ShowPage(p.id)
}

func (p *timedPopup) Hide() {
	thunder.Pages.RemovePage(p.id)
	p.Close()
}

func (p *timedPopup) Primitive() tview.Primitive {
	return p.primitive
}

func (p *timedPopup) Focusable() bool {
	return p.focusable
}

func (p *timedPopup) Close() {
	p.once.Do(func() {
		if p.hide != nil {
			close(p.hide)
		}
	})
}

type stack struct {
	Push   func(Popup)
	Pop    func() Popup
	Top    func() Popup
	Length func() int
}

func newStack() stack {
	slice := make([]Popup, 0)

	return stack{
		Push: func(p Popup) {
			slice = append(slice, p)
		},
		Pop: func() Popup {
			res := slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			return res
		},
		Top: func() Popup {
			return slice[len(slice)-1]
		},
		Length: func() int {
			return len(slice)
		},
	}
}

type PopupManager struct {
	prevTimedPopup Popup
	stack          stack
	mu             sync.Mutex
}

func NewPopupManager() *PopupManager {
	return &PopupManager{
		stack: newStack(),
	}
}

func (pm *PopupManager) Show(p Popup) {

	switch popup := p.(type) {
	case *defaultPopup:
		popupCount.Add(1)
		pm.stack.Push(popup)

		positionFunc := popup.position.ToFunc()
		thunder.Pages.AddPage(popup.id, positionFunc(popup.primitive, p.Width(), p.Height()), true, false)

		popup.Show()
		if popup.focusable {
			thunder.UnfocusAll()
			thunder.SetFocusPanel(nil, popup.primitive.(Panel)) // TODO
		}

	case *timedPopup:
		prev := pm.prevTimedPopup

		if prev != nil {
			prev.(*timedPopup).Close()
		}

		popupCount.Add(1)

		positionFunc := popup.position.ToFunc()
		thunder.Pages.AddPage(popup.id, positionFunc(popup.primitive, p.Width(), p.Height()), true, false)

		popup.Show()

		thunder.ResetFocus()

		pm.prevTimedPopup = popup
		pm.stack.Push(popup)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), popup.timeout)
			defer cancel()

			select {
			case <-ctx.Done(): // Timeout
			case <-popup.hide: // New popup added -> remove old one
			}

			thunder.App.QueueUpdateDraw(func() {
				pm.Hide(popup)
			})
		}()
	}
}

func (pm *PopupManager) Hide(popup Popup) {
	popup.Hide()
	pm.SwitchFocus(popup)
}

func (pm *PopupManager) SwitchFocus(prev Popup) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.stack.Length() == 0 {
		thunder.ResetFocus()
		return
	}

	curr := pm.stack.Pop()

	prevPanel := prev.Primitive().(Panel)

	var currPanel Panel

	if !curr.Focusable() || curr == prev {
		thunder.ResetFocus()
		return
	}

	currPanel = curr.Primitive().(Panel)
	thunder.SetFocusPanel(prevPanel, currPanel)
}

func confirmationPopup(text string, handler func(int, string)) {
	id := fmt.Sprintf("confirmation_popup-%d", popupCount.Load())

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

	popup := NewDefaultPopup(modal, id, true, Center, 0, 0)

	modal.SetDoneFunc(func(indx int, label string) {
		thunder.Popups.Hide(popup)
		handler(indx, label)
	})

	thunder.Popups.Show(popup)
}

func errorPopup(err error) {
	timeoutPopup(err.Error(), 2*time.Second, tcell.ColorRed, tcell.ColorRed)
}

func infoPopup(text string) {
	timeoutPopup(text, 2*time.Second, thunder.Colors.Accent, thunder.Colors.Foreground)
}

func timeoutPopup(text string, timeout time.Duration, textColor, borderColor tcell.Color) {
	id := fmt.Sprintf("timeout_popup-%d", popupCount.Load())

	textView := tview.NewTextView().
		SetText(text).
		SetTextColor(textColor).SetTextAlign(tview.AlignCenter)

	textView.SetBackgroundColor(thunder.Colors.Background).SetBorder(true).
		SetBorderColor(borderColor).SetBorderPadding(1, 1, 1, 1)

	popup := NewTimeoutPopup(textView, id, false, TopRight, timeout, 0, 0)

	thunder.Popups.Show(popup)
}

// zero width or height means primitive [width or height] will be resized automatically
func topRight(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 20, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 2, false).
			AddItem(nil, 0, 8, false), width, 15, false).
		AddItem(nil, 0, 1, false)
}

// zero width or height means primitive [width or height] will be resized automatically
func center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(p, width, 3, false).
			AddItem(nil, 0, 1, false), height, 3, false).
		AddItem(nil, 0, 1, false)
}

func downloadPopup() {
	colors := thunder.Colors
	textStyle := tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(colors.Background)
	placeholderStyle := tcell.StyleDefault.Foreground(tcell.ColorGrey).Background(colors.Background)

	input := NewInput("Name of the song...", textStyle, placeholderStyle)

	list := tview.NewList()

	title := "Download songs"

	flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(input, 1, 0, true).AddItem(tview.NewBox(), 1, 0, false).AddItem(list, 0, 1, false)

	frame := tview.NewFrame(flex)

	var focused int

	const (
		inputPanel int = iota
		listPanel
	)

	focus := func(focus int) {
		if focus == inputPanel {
			thunder.SetFocusPanel(list, input)
		} else {
			thunder.SetFocusPanel(input, list)
		}

		focused = focus
	}

	cycle := func() {
		if focused == inputPanel {
			focus(listPanel)
			focused++
		} else {
			focus(inputPanel)
			focused = 0
		}
	}

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

	var songs []downloadInfo
	currentIdx := 0
	requesting := false
	downloading := false

	getSongs := func(query string) {
		defer func() { requesting = false }()

		format := fmt.Sprintf("ytsearch%d:%s", 20, query)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, "yt-dlp", format, "--flat-playlist", "--dump-single-json")

		result := make(chan []byte, 1)
		var outErr error
		go func() {
			thunder.App.QueueUpdateDraw(func() {
				frame.SetTitle("Fetching...")
			})

			output, err := cmd.Output()
			if err != nil {
				outErr = fmt.Errorf("execute command: %w", err)
			}
			result <- output
		}()

		go func() {
			select {
			case output := <-result:
				list.Clear()
				if outErr != nil {
					errorPopup(outErr)
					SendErr(outErr)
					return
				}

				var result searchResult

				if err := json.Unmarshal(output, &result); err != nil {
					errorPopup(err)
					SendErr(err)
					return
				}

				songs = make([]downloadInfo, 0, len(result.Entries))

				for i, entry := range result.Entries {
					info := downloadInfo{
						url:   "https://youtube.com/watch?v=" + entry.ID,
						title: entry.Title,
					}

					if entry.Duration != nil {
						info.duration = *entry.Duration
					}

					songs = append(songs, info)

					var duration string

					h, m, s := info.duration/3600, info.duration/60%60, info.duration%60
					if h > 0 {
						duration = fmt.Sprintf("%d:%02d:%02d", h, m, s)
					} else {
						duration = fmt.Sprintf("%d:%02d", m, s)
					}

					list.InsertItem(i, fmt.Sprintf("%s %s", info.title, duration), "", 0, nil)
				}
				focus(listPanel)
				cancel()
			}
			thunder.App.QueueUpdateDraw(func() {
				frame.SetTitle(title)
			})
			requesting = false
		}()

	}

	download := func(song downloadInfo) {
		defer func() { downloading = false }()
		frame.SetTitle("Starting download...")
		cmd := exec.Command(
			"yt-dlp",
			"-x",
			"--no-playlist",
			"--audio-format", "mp3",
			"--audio-quality", "0",
			"--newline",
			"--progress-template",
			"downloading%(progress.downloaded_bytes)s %(progress.total_bytes)s %(progress.total_bytes_estimate)s %(progress.speed)s %(progress.eta)s",
			"-o", thunder.Library.GetCurrentPath()+"/%(title)s.%(ext)s",
			song.url,
		)

		reader, writer, err := os.Pipe()
		if err != nil {
			errorPopup(err)
			SendErr(err)
			return
		}

		cmd.Stdout = writer
		cmd.Stderr = writer

		if err := cmd.Start(); err != nil {
			reader.Close()
			writer.Close()
			errorPopup(err)
			SendErr(err)
			return
		}

		writer.Close()

		scanner := bufio.NewScanner(reader)
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

			if total == 0 {
				total, _ = strconv.ParseInt(fields[2], 10, 64)
			}

			speed, _ := strconv.ParseFloat(fields[3], 64)
			eta, _ := strconv.Atoi(fields[4])

			percent := float64(downloaded) / float64(total) * 100
			title := fmt.Sprintf(
				"─Downloading──┤ %.1f%% / %.2f MB/s │ ETA %ds ├",
				percent,
				speed/1024/1024,
				eta,
			)

			thunder.App.QueueUpdateDraw(func() {
				frame.SetTitle(title)
			})

			thunder.Logger.Info(title)
		}

		scanErr := scanner.Err()
		reader.Close()

		waitErr := cmd.Wait()
		if scanErr != nil {
			errorPopup(err)
			SendErr(err)
			return
		}

		if waitErr != nil {
			errorPopup(err)
			SendErr(err)
			return
		}

		thunder.App.QueueUpdateDraw(func() {
			frame.SetTitle("Download completed")
		})

	}

	input.SetBorderColor(colors.Accent).SetBackgroundColor(colors.Background)
	input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEnter {
			if requesting {
				return e
			}

			query := input.GetText()
			if strings.TrimSpace(query) != "" {
				requesting = true
				getSongs(query)
			}
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
		switch e.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		case ' ':
			if len(songs) == 0 || downloading {
				return nil
			}

			song := songs[currentIdx]

			downloading = true
			go download(song)
		}

		return nil
	})

	list.SetChangedFunc(func(idx int, _, _ string, _ rune) {
		currentIdx = idx
	})

	flex.SetBorderPadding(0, 0, 1, 1)

	frame.SetTitle(title).SetTitleColor(colors.Accent).SetBorder(true).SetBorderColor(colors.Accent)

	id := fmt.Sprintf("download_popup-%d", popupCount.Load())

	popup := NewDefaultPopup(frame, id, true, Center, 0, 0)

	flex.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEsc:
			thunder.Popups.Hide(popup)
		case tcell.KeyTab:
			cycle()
		}
		return e
	})

	thunder.Popups.Show(popup)
}

func inputPopup(title, placeholder, text string, handler func(ok bool, text string)) {
	colors := thunder.Colors
	id := fmt.Sprintf("confirmation_popup-%d", popupCount.Load())
	textStyle := tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(colors.Background)
	placeholderStyle := tcell.StyleDefault.Foreground(tcell.ColorGrey).Background(colors.Background)

	input := NewInput(placeholder, textStyle, placeholderStyle).SetText(text)

	popup := NewDefaultPopup(input, id, true, Center, 0, 5)

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			handler(false, "")
		case tcell.KeyEnter:
			handler(true, input.GetText())
		}
		thunder.Popups.Hide(popup)
	})

	input.SetBorder(true).SetBorderColor(colors.Foreground).SetTitle(title).SetBorderPadding(1, 1, 1, 1)

	thunder.Popups.Show(popup)

}

func NewInput(placeholder string, textStyle, placeholderStyle tcell.Style) *tview.InputField {
	return tview.NewInputField().SetPlaceholder(placeholder).SetFieldStyle(textStyle).SetPlaceholderStyle(placeholderStyle)
}
