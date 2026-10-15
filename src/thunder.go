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
	Panels          []Panel
	Popups          *PopupManager
	Colors          *Colors
	CurrentPanelIdx int
}

func (t *Thunder) InitPanels(app *tview.Application) {
	home, err := os.UserHomeDir()
	if err != nil {
		exit(err)
	}
	playlistInfo := NewDefaultPlaylistInfo(home)
	t.Player = NewPlayer()
	t.Library = NewLibrary(home)
	t.PlayPanel = NewPlayPanel(playlistInfo)
	t.PlayBar = NewPlayBar()
	t.Pages = tview.NewPages()
	t.App = app
	t.Panels = []Panel{t.Library, t.PlayPanel, t.PlayBar}
	t.Popups = NewPopupManager()
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
