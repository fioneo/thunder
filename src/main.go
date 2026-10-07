package main

import (
	// "log"
	// "time"
	//
	// "os"
	// "fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func InitApp(app *tview.Application) {
	thunder = NewThunder()
	thunder.InitPanels(app)
	flex := tview.NewFlex().
		AddItem(thunder.Library, 0, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(thunder.PlayPanel, 0, 7, false).AddItem(thunder.PlayBar, 0, 3, false), 0, 3, false)

	thunder.Pages.AddPage("main", flex, true, true)
	thunder.App.SetRoot(thunder.Pages, true).SetFocus(thunder.Library)

	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if thunder.Pages.HasPage("confirmation_popup") {
			return e
		}
		if e.Key() == tcell.KeyTab {
			thunder.CyclePanels()
		}
		return e
	})
	tview.Styles.PrimitiveBackgroundColor = thunder.Colors.Background
}

func main() {

	app := tview.NewApplication()
	InitApp(app)

	if err := app.Run(); err != nil {
		exit(err)
	}
}
