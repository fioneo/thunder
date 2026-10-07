package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func confirmationPopup(text string, handler func(int, string)) {
	currentPanel := thunder.Panels[thunder.CurrentPanelIdx]
	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(thunder.Colors.Background).
		AddButtons([]string{"no", "yes"}).
		SetButtonBackgroundColor(thunder.Colors.Background).
		SetButtonTextColor(thunder.Colors.Accent)

	modal.SetDoneFunc(func(indx int, label string) {
		thunder.Pages.RemovePage("confirmation_popup")
		thunder.SetFocusPanel(modal, currentPanel)
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
	thunder.SetFocusPanel(currentPanel, modal)
}
func errorPopup(err error) {
	currentPanel := thunder.Panels[thunder.CurrentPanelIdx]
	modal := tview.NewModal().
		SetText(err.Error()).
		SetTextColor(tcell.ColorRed).
		SetBackgroundColor(thunder.Colors.Background).
		AddButtons([]string{"ok"}).
		SetButtonBackgroundColor(thunder.Colors.Background).
		SetButtonTextColor(tcell.ColorRed).
		SetBackgroundColor(thunder.Colors.Background)

	modal.SetDoneFunc(func(indx int, label string) {
		thunder.Pages.RemovePage("error_popup")
		thunder.SetFocusPanel(modal, currentPanel)
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

	thunder.Pages.AddPage("error_popup", modal, true, true)
	thunder.SetFocusPanel(currentPanel, modal)
	modal.SetBorderColor(tcell.ColorRed)
}
