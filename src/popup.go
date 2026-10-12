package main

import (
	"context"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sync/atomic"
	"time"
)

var (
	popupCount atomic.Uint32
)

type popupEntry struct {
	panel     Panel
	page      string
	focusable bool
}

type PopupQueue struct {
	popups []popupEntry
}

func NewPopupQueue() *PopupQueue {
	return &PopupQueue{}
}

func (q *PopupQueue) Enqueue(popup tview.Primitive, page string, focusable bool) {
	panel, ok := popup.(Panel)
	if !ok {
		return
	}

	q.popups = append(q.popups, popupEntry{
		panel:     panel,
		page:      page,
		focusable: focusable,
	})
	popupCount.Add(1)

	if q.Length() == 1 {
		if focusable {
			thunder.UnfocusAll()
		}
		q.Next()
	}

}

func (q *PopupQueue) Next() {
	if q.Length() == 0 {
		q.Reset()
		return
	}
	next := q.popups[0]

	if next.focusable {
		thunder.SetFocusPanel(nil, next.panel)
	}
	thunder.Pages.ShowPage(next.page)
}

func (q *PopupQueue) DismissCurrent() {
	if q.Length() == 0 {
		return
	}

	popup := q.popups[0]
	thunder.Pages.RemovePage(popup.page)

	q.popups = q.popups[1:]
	q.Next()
}

func (q *PopupQueue) Length() int {
	return len(q.popups)
}

func (q *PopupQueue) Reset() {
	thunder.SetFocusPanel(nil, thunder.Panels[thunder.CurrentPanelIdx])
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
	thunder.Popups.Enqueue(modal, page, true)
}

func errorPopup(err error) {
	timeoutPopup(err.Error(), 3*time.Second)
}

func timeoutPopup(title string, timeout time.Duration) {
	page := fmt.Sprintf("timeout_popup-%d", popupCount.Load())

	textView := tview.NewTextView().
		SetText(title).
		SetTextColor(thunder.Colors.Accent).SetTextAlign(tview.AlignCenter)

	textView.SetBackgroundColor(thunder.Colors.Background).SetBorder(true).
		SetBorderColor(thunder.Colors.Foreground).SetBorderPadding(1, 1, 1, 1)

	thunder.Pages.AddPage(page, topRight(textView), true, false)
	thunder.Popups.Enqueue(textView, page, false)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		<-ctx.Done()
		thunder.App.QueueUpdateDraw(func() {
			thunder.Popups.DismissCurrent()
		})
	}()
}

func topRight(p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 20, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, 0, 4, true).
			AddItem(nil, 0, 8, false), 0, 12, true).
		AddItem(nil, 0, 1, false)
}
