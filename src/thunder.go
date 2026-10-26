package main

import (
	"github.com/fioneo/thunder/src/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"sync"
)

var thunder *Thunder

type Thunder struct {
	App     *tview.Application
	Pages   *tview.Pages
	Library *Library
	// PlayPanel       *PlayPanel
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
	home, err := os.UserHomeDir()
	if err != nil {
		exit(err)
	}

	t.App = app
	t.Pages = tview.NewPages()
	t.Popups = NewPopupManager()
	vol := percentageToVolume(args.volume)
	t.Playback = NewPlaybackController(NewPlayer(vol))

	t.Library = NewLibrary(home)
	t.Queue = NewQueueView(t.Playback)
	t.PlayBar = NewPlayBar(t.Playback.Progress)
	t.Panels = []Panel{t.Library, t.Queue, t.PlayBar}

	t.Playback.OnChange(func(s Snapshot) {
		go app.QueueUpdateDraw(func() { t.renderPlayback(s) })
	})
	t.Playback.OnError(func(err error) {
		SendErr(err)
		go app.QueueUpdateDraw(func() {
			errorPopup(err)
		})
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

func (t *Thunder) SendErr(err error) {
	if err == nil {
		return
	}
	select {
	case t.ErrCh <- err:
	default:
	}
}

func NewThunder() *Thunder {

	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
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

func (t *Thunder) SetFocusPanel(prevPanel, currentPanel Panel) {

	if prevPanel != nil {
		prevPanel.SetBorderColor(t.Colors.Foreground)
		prevPanel.SetTitleColor(t.Colors.Foreground)
	}

	if currentPanel != nil {
		primitive := currentPanel.(tview.Primitive)

		t.App.SetFocus(primitive)

		currentPanel.SetBorderColor(t.Colors.Accent)
		currentPanel.SetTitleColor(t.Colors.Accent)
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
	thunder.SetFocusPanel(nil, thunder.Panels[thunder.CurrentPanelIdx])
}

func (t *Thunder) Stop() {
	t.stopOnce.Do(
		func() {
			thunder.Logger.Info("Stopping application")

			thunder.Playback.Close()
			thunder.Server.Close()
			thunder.Logger.Close()
			thunder.App.Stop()
		},
	)
}
