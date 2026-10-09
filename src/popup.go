package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sync/atomic"
)

var (
	popupCount atomic.Uint32
)

type popupEntry struct {
	panel Panel
	page  string
}

type PopupQueue struct {
	popups []popupEntry
}

func NewPopupQueue() *PopupQueue {
	return &PopupQueue{}
}

func (q *PopupQueue) Enqueue(popup tview.Primitive, page string) {
	panel, ok := popup.(Panel)
	if !ok {
		return
	}

	q.popups = append(q.popups, popupEntry{
		panel: panel,
		page:  page,
	})
	popupCount.Add(1)

	if q.Length() == 1 {
		q.Next()
	}

}

func (q *PopupQueue) Next() {
	if q.Length() == 0 {
		thunder.App.SetFocus(thunder.Panels[thunder.CurrentPanelIdx].(tview.Primitive))
		return
	}
	next := q.popups[0]

	thunder.App.SetFocus(next.panel.(tview.Primitive))
	thunder.Pages.ShowPage(next.page)
}

func (q *PopupQueue) DismissCurrent() {
	if q.Length() == 0 {
		return
	}

	popup := q.popups[0]
	thunder.Pages.RemovePage(popup.page)

	thunder.App.SetFocus(popup.panel.(tview.Primitive))

	q.popups = q.popups[1:]
	q.Next()
}

func (q *PopupQueue) Length() int {
	return len(q.popups)
}

func confirmationPopup(text string, handler func(int, string)) {
	page := fmt.Sprintf("confirmation_popup-%d", popupCount.Load())

	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(thunder.Colors.Background).
		AddButtons([]string{"no", "yes"}).
		SetButtonStyle(tcell.StyleDefault.Foreground(thunder.Colors.Accent).Background(thunder.Colors.Background)).
		SetButtonActivatedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Accent))

	modal.SetDoneFunc(func(indx int, label string) {
		thunder.Popups.DismissCurrent()
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

	thunder.Pages.AddPage(page, modal, true, false)
	thunder.Popups.Enqueue(modal, page)
}
func errorPopup(err error) {
	page := fmt.Sprintf("error_popup-%d", popupCount.Load())

	modal := tview.NewModal().
		SetText(err.Error()).
		SetTextColor(tcell.ColorRed).
		SetBackgroundColor(thunder.Colors.Background).
		AddButtons([]string{"ok"}).
		SetButtonStyle(tcell.StyleDefault.Foreground(thunder.Colors.Accent).Background(thunder.Colors.Background)).
		SetButtonActivatedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Accent))

	modal.SetDoneFunc(func(_ int, _ string) {
		thunder.Popups.DismissCurrent()
	})

	modal.SetBorderColor(tcell.ColorRed).
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

	thunder.Pages.AddPage(page, modal, true, false)
	thunder.Popups.Enqueue(modal, page)
}
