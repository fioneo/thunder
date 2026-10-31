package main

import (
	"fmt"
	"github.com/fioneo/thunder/src/config"
	"github.com/fioneo/thunder/src/logger"
	"github.com/fioneo/thunder/src/player"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sync"
	"time"
)

var thunder *Thunder
var colors config.Colors

type Thunder struct {
	App             *tview.Application
	Pages           *tview.Pages
	Library         *Library
	Queue           *QueueView
	PlayBar         *PlayBar
	Playback        *PlaybackController
	Panels          []Panel
	Popups          *PopupManager
	Server          *Server
	Logger          *logger.Logger
	Config          *config.Config
	ErrCh           chan error
	CurrentPanelIdx int
	stopOnce        sync.Once
}

func (t *Thunder) InitPanels(app *tview.Application, args Args) {
	t.App = app
	t.Pages = tview.NewPages()
	t.Popups = NewPopupManager()

	vol := percentageToVolume(args.volume)
	t.Playback = NewPlaybackController(player.New(vol))

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
		err := fmt.Errorf("Initialize server: %w", err)
		exit(err)
	}

	t.Server = server
}

func (t *Thunder) InitLogger(args Args) {
	logger, err := logger.New(args.logPath)
	if err != nil {
		err := fmt.Errorf("Initialize logger: %w", err)
		exit(err)
	}
	t.Logger = logger
}

func (t *Thunder) ParseConfig(args Args) {
	config, err := config.Parse(args.cfgPath)
	if err != nil {
		err := fmt.Errorf("Parse config: %w", err)
		exit(err)
	}
	t.Config = config
}

func (t *Thunder) renderPlayback(s Snapshot) {
	t.Queue.Render(s)
	t.PlayBar.Render(s)
	t.Server.Write(s)
}

func NewThunder() *Thunder {
	return &Thunder{
		ErrCh: make(chan error, 64),
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

	t.Unfocus(prevPanel)
	t.Focus(currentPanel)
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
		c.ChangeVolume(player.VolumeStep)
	case '-':
		c.ChangeVolume(-player.VolumeStep)
	case 'm':
		c.ToggleMute()
	case 'y':
		c.CycleMode()
	default:
		return e
	}
	return nil
}

func (t *Thunder) Unfocus(panel Panel) {
	if panel != nil {
		panel.SetBorderColor(colors.Foreground)
		panel.SetTitleColor(colors.Foreground)
	}
}

func (t *Thunder) Focus(panel Panel) {
	if panel != nil {
		primitive := panel.(tview.Primitive)

		t.App.SetFocus(primitive)

		panel.SetBorderColor(colors.Accent)
		panel.SetTitleColor(colors.Accent)
	}
}

func (t *Thunder) UnfocusAll() {
	for _, p := range t.Panels {
		p.SetBorderColor(colors.Foreground)
		p.SetTitleColor(colors.Foreground)
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
