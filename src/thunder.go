package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
)

var thunder *Thunder

type Thunder struct {
	App             *tview.Application
	Pages           *tview.Pages
	Library         *Library
	PlayPanel       *PlayPanel
	PlayBar         *PlayBar
	Player          *Player
	Colors          *Colors
	Panels          []Panel
	CurrentPanelIdx int
}

func (t *Thunder) InitPanels(app *tview.Application) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		exit(err)
	}
	playlistInfo := NewDefaultPlaylistInfo()
	t.Player = NewPlayer()
	t.Library = NewLibrary(homeDir)
	t.PlayPanel = NewPlayPanel(playlistInfo)
	t.PlayBar = NewPlayBar()
	t.Pages = tview.NewPages()
	t.App = app
	t.Panels = []Panel{t.Library, t.PlayPanel, t.PlayBar}
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
	}
}

type Colors struct {
	Accent     tcell.Color
	Song       tcell.Color
	Playlist   tcell.Color
	Background tcell.Color
	Foreground tcell.Color
	Popup      tcell.Color
}

func DefaultColors() *Colors {
	return &Colors{
		Background: tcell.ColorDefault,
		Foreground: tcell.GetColor("#E5E9F0"),
		Accent:     tcell.GetColor("#88C0D0"),
		Song:       tcell.GetColor("#81A1C1"),
		Playlist:   tcell.GetColor("#88C0D0"),
		Popup:      tcell.GetColor("#7BB3C7"),
	}
}

func (t *Thunder) CyclePanels() {
	if len(t.Panels) == 0 {
		return
	}
	prevPanel := t.Panels[t.CurrentPanelIdx]
	t.CurrentPanelIdx++

	if t.CurrentPanelIdx >= len(t.Panels) {
		t.CurrentPanelIdx = 0
	}

	currentPanel := t.Panels[t.CurrentPanelIdx]

	t.SetFocusPanel(prevPanel, currentPanel)
}

func (t *Thunder) SetFocusPanel(prevPanel, currentPanel Panel) {

	primitive := currentPanel.(tview.Primitive)

	t.App.SetFocus(primitive)

	prevPanel.SetBorderColor(t.Colors.Foreground)
	prevPanel.SetTitleColor(t.Colors.Foreground)

	currentPanel.SetBorderColor(t.Colors.Accent)
	currentPanel.SetTitleColor(t.Colors.Accent)
}
