package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func confirmationPopup(text string, handler func(int, string)) {
	tview.Styles.PrimitiveBackgroundColor = thunder.Colors.Background
	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(thunder.Colors.Background).
		AddButtons([]string{"no", "yes"}).
		SetButtonBackgroundColor(thunder.Colors.Background).
		SetButtonTextColor(thunder.Colors.Accent).
		SetDoneFunc(func(indx int, label string) {
			thunder.Pages.RemovePage("confirmation_popup")
			handler(indx, label)
		})

	modal.SetBorderColor(thunder.Colors.Accent).
		SetBackgroundColor(thunder.Colors.Background)
	modal.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		case ' ':
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		}

		return e
	})

	thunder.Pages.AddPage("confirmation_popup", modal, true, true)
	thunder.App.SetFocus(modal)
}
