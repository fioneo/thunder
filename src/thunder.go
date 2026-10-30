package main

import (
	"github.com/fioneo/thunder/src/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sync"
	"time"
)

var thunder *Thunder

type Thunder struct {
	App             *tview.Application
	Pages           *tview.Pages
	Library         *Library
	Queue           *QueueView
	PlayBar         *PlayBar
	Playback        *PlaybackController
	Panels          []Panel
	Popups          *PopupManager
	Colors          *Colors
	Server          *Server
	Logger          *logger.Logger
	ErrCh           chan error
	CurrentPanelIdx int
	stopOnce        sync.Once
}

func (t *Thunder) InitPanels(app *tview.Application, args Args) {
	t.App = app
	t.Pages = tview.NewPages()
	t.Popups = NewPopupManager()
	vol := percentageToVolume(args.volume)
	t.Playback = NewPlaybackController(NewPlayer(vol))

	t.Library = NewLibrary(args.musicPath)
	t.Queue = NewQueueView(t.Playback)
	t.PlayBar = NewPlayBar(t.Playback.Progress)
	t.Panels = []Panel{t.Library, t.Queue, t.PlayBar}

	t.Playback.OnChange(func(s Snapshot) {
		go onUI(func() { t.renderPlayback(s) })
	})
	t.Playback.OnError(func(err error) {
		SendErr(err)
	})
}

func (t *Thunder) InitServer(args Args) {
	server, err := NewServer(args.sockPath, t.Playback)
	if err != nil {
		exit(err)
	}

	t.Server = server
}

func (t *Thunder) InitLogger(args Args) {
	logger, err := logger.New(args.logPath)
	if err != nil {
		exit(err)
	}
	t.Logger = logger
}

func (t *Thunder) renderPlayback(s Snapshot) {
	t.Queue.Render(s)
	t.PlayBar.Render(s)
	t.Server.Write(s)
}

func NewThunder() *Thunder {
	return &Thunder{
		Colors: DefaultColors(),
		ErrCh:  make(chan error, 64),
	}
}

type Colors struct {
	Accent      tcell.Color
	Song        tcell.Color
	Playlist    tcell.Color
	Background  tcell.Color
	Foreground  tcell.Color
	Popup       tcell.Color
	Selected    tcell.Color
	Highlighted tcell.Color
}

func DefaultColors() *Colors {
	return &Colors{
		Background:  tcell.ColorDefault,
		Foreground:  tcell.GetColor("#FEF9FF"),
		Accent:      tcell.GetColor("#6DB2EE"),
		Song:        tcell.GetColor("#7CA7DF"),
		Playlist:    tcell.GetColor("#6DB2EE"),
		Popup:       tcell.GetColor("#6DB2EE"),
		Selected:    tcell.GetColor("#9F9FED"),
		Highlighted: tcell.GetColor("#59BDEB"),
	}
}

func (t *Thunder) CyclePanels(forward bool) {
	if len(t.Panels) == 0 {
		return
	}

	prevPanel := t.Panels[t.CurrentPanelIdx]

	if forward {
		t.CurrentPanelIdx++
		if t.CurrentPanelIdx >= len(t.Panels) {
			t.CurrentPanelIdx = 0
		}
	} else {
		t.CurrentPanelIdx--
		if t.CurrentPanelIdx < 0 {
			t.CurrentPanelIdx = len(t.Panels) - 1
		}
	}

	currentPanel := t.Panels[t.CurrentPanelIdx]

	t.SetFocusPanel(prevPanel, currentPanel)
}

const seekStep = 5 * time.Second

func (t *Thunder) handlePlaybackKey(e *tcell.EventKey) *tcell.EventKey {
	if e.Key() != tcell.KeyRune {
		return e
	}

	c := t.Playback
	switch e.Rune() {
	case 'p':
		c.TogglePause()
	case '>':
		c.Next()
	case '<':
		c.Previous()
	case 'x':
		c.Stop()
	case ']':
		c.Seek(seekStep)
	case '[':
		c.Seek(-seekStep)
	case '+', '=':
		c.ChangeVolume(volumeStep)
	case '-':
		c.ChangeVolume(-volumeStep)
	case 'm':
		c.ToggleMute()
	case 'y':
		c.CycleMode()
	default:
		return e
	}
	return nil
}

func (t *Thunder) SetFocusPanel(prevPanel, currentPanel Panel) {
	t.Unfocus(prevPanel)
	t.Focus(currentPanel)
}

func (t *Thunder) Unfocus(panel Panel) {
	if panel != nil {
		panel.SetBorderColor(t.Colors.Foreground)
		panel.SetTitleColor(t.Colors.Foreground)
	}
}

func (t *Thunder) Focus(panel Panel) {
	if panel != nil {
		primitive := panel.(tview.Primitive)

		t.App.SetFocus(primitive)

		panel.SetBorderColor(t.Colors.Accent)
		panel.SetTitleColor(t.Colors.Accent)
	}
}

func (t *Thunder) UnfocusAll() {
	for _, p := range t.Panels {
		p.SetBorderColor(t.Colors.Foreground)
		p.SetTitleColor(t.Colors.Foreground)
	}
}

func (t *Thunder) ResetFocus() {
	t.UnfocusAll()
	t.Focus(t.Panels[t.CurrentPanelIdx])
}

func (t *Thunder) Stop() {
	t.stopOnce.Do(
		func() {
			t.Logger.Info("Stopping application")

			t.Playback.Close()
			t.Server.Close()
			t.Logger.Close()
			t.App.Stop()
		},
	)
}
