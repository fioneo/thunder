package main

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const redrawInterval = 500 * time.Millisecond

func InitApp(app *tview.Application) {
	thunder = NewThunder()
	thunder.InitPanels(app)
	flex := tview.NewFlex().
		AddItem(thunder.Library, 0, 2, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(thunder.PlayPanel, 0, 7, false).AddItem(thunder.PlayBar, 0, 3, false), 0, 6, false)

	thunder.Pages.AddPage("main", flex, true, true)
	thunder.App.SetRoot(thunder.Pages, true).SetFocus(thunder.Library)

	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		pageName, _ := thunder.Pages.GetFrontPage()
		if pageName != "main" && !strings.Contains(pageName, "timeout_popup") {
			return e
		}
		if e.Key() == tcell.KeyTab {
			thunder.CyclePanels()
		}
		return thunder.handlePlaybackKey(e)
	})
	tview.Styles.PrimitiveBackgroundColor = thunder.Colors.Background
}

func redrawWhilePlaying(app *tview.Application, done <-chan struct{}) {
	ticker := time.NewTicker(redrawInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if thunder.Playback.Snapshot().Active() {
				app.QueueUpdateDraw(func() {})
			}
		}
	}
}

func main() {
	app := tview.NewApplication()
	InitApp(app)

	done := make(chan struct{})
	go redrawWhilePlaying(app, done)

	err := app.Run()

	close(done)
	thunder.Playback.Close()
	thunder.Server.Close()

	if err != nil {
		exit(err)
	}
}
