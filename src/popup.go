package main

import (
	"context"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sync"
	"sync/atomic"
	"time"
)

type Position uint8

const (
	Center Position = iota
	TopRight
)

func (p Position) ToFunc() func(tview.Primitive, int, int) tview.Primitive {
	var (
		f func(tview.Primitive, int, int) tview.Primitive
	)
	switch p {
	case Center:
		f = center
	case TopRight:
		f = topRight
	}
	return f
}

var (
	popupCount atomic.Uint32
)

type defaultPopup struct {
	id        string
	primitive tview.Primitive
	focusable bool
	position  Position
}

type timedPopup struct {
	*defaultPopup
	timeout time.Duration
	hide    chan struct{}
	once    sync.Once
}

func NewDefaultPopup(p tview.Primitive, id string, focusable bool, position Position) *defaultPopup {
	popup := &defaultPopup{
		primitive: p,
		id:        id,
		focusable: focusable,
		position:  position,
	}

	return popup
}

func NewTimeoutPopup(p tview.Primitive, id string, focusable bool, position Position, timeout time.Duration) *timedPopup {
	return &timedPopup{
		defaultPopup: &defaultPopup{
			primitive: p,
			id:        id,
			focusable: focusable,
			position:  position,
		},
		hide:    make(chan struct{}),
		timeout: timeout,
	}
}

func (p *defaultPopup) ID() string {
	return p.id
}

func (p *defaultPopup) Show() {
	thunder.Pages.ShowPage(p.id)
}

func (p *defaultPopup) Hide() {
	thunder.Pages.RemovePage(p.id)
}

func (p *defaultPopup) Primitive() tview.Primitive {
	return p.primitive
}

func (p *defaultPopup) Focusable() bool {
	return p.focusable
}

func (p *timedPopup) ID() string {
	return p.id
}

func (p *timedPopup) Show() {
	thunder.Pages.ShowPage(p.id)
}

func (p *timedPopup) Hide() {
	thunder.Pages.RemovePage(p.id)
}

func (p *timedPopup) Primitive() tview.Primitive {
	return p.primitive
}

func (p *timedPopup) Focusable() bool {
	return p.focusable
}

func (p *timedPopup) Close() {
	p.once.Do(func() {
		if p.hide != nil {
			close(p.hide)
		}
	})
}

type stack struct {
	Push   func(Popup)
	Pop    func() Popup
	Top    func() Popup
	Length func() int
}

func newStack() stack {
	slice := make([]Popup, 0)

	return stack{
		Push: func(p Popup) {
			slice = append(slice, p)
		},
		Pop: func() Popup {
			res := slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			return res
		},
		Top: func() Popup {
			return slice[len(slice)-1]
		},
		Length: func() int {
			return len(slice)
		},
	}
}

type PopupManager struct {
	popups         map[string]Popup
	prevTimedPopup Popup
	stack          stack
	mu             sync.RWMutex
}

func NewPopupManager() *PopupManager {
	return &PopupManager{
		popups: make(map[string]Popup),
		stack:  newStack(),
	}
}

func (pm *PopupManager) Show(p Popup) {

	switch popup := p.(type) {
	case *defaultPopup:
		popupCount.Add(1)
		pm.stack.Push(popup)

		positionFunc := popup.position.ToFunc()
		thunder.Pages.AddPage(popup.id, positionFunc(popup.primitive, 70, 7), true, false)

		popup.Show()
		if popup.focusable {
			thunder.UnfocusAll()
			thunder.SetFocusPanel(nil, popup.primitive.(Panel)) // TODO
		}

	case *timedPopup:
		prev := pm.prevTimedPopup

		if prev != nil {
			prev.(*timedPopup).Close()
		}

		popupCount.Add(1)
		pm.stack.Push(popup)

		positionFunc := popup.position.ToFunc()
		thunder.Pages.AddPage(popup.id, positionFunc(popup.primitive, 70, 7), true, false)

		if popup.focusable {
			thunder.SetFocusPanel(nil, popup.primitive.(Panel)) // TODO
		}

		popup.Show()
		thunder.ResetFocus()

		pm.prevTimedPopup = popup
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), popup.timeout)
			defer cancel()

			select {
			case <-ctx.Done(): // Timeout
			case <-popup.hide: // New popup added -> remove old one
			}

			thunder.App.QueueUpdateDraw(func() {
				pm.Hide(popup)
			})
		}()
	}
	pm.popups[p.ID()] = p
}

func (pm *PopupManager) Hide(popup Popup) {
	popup.Hide()
	pm.SwitchFocus(popup)
	delete(pm.popups, popup.ID())
}

func (pm *PopupManager) SwitchFocus(prev Popup) {
	pm.mu.Lock()
	if pm.stack.Length() == 0 {
		thunder.ResetFocus()
		pm.mu.Unlock()
		return
	}

	popup := pm.stack.Pop()

	if !popup.Focusable() {
		pm.mu.Unlock()
		pm.SwitchFocus(popup)
		return
	}

	thunder.SetFocusPanel(prev.Primitive().(Panel), popup.Primitive().(Panel))
	pm.mu.Unlock()
}

func (pm *PopupManager) Get(id string, popup Popup) {}

func confirmationPopup(text string, handler func(int, string)) {
	id := fmt.Sprintf("confirmation_popup-%d", popupCount.Load())

	modal := tview.NewModal().
		SetText(text).
		SetBackgroundColor(thunder.Colors.Background).
		AddButtons([]string{"no", "yes"}).
		SetButtonStyle(tcell.StyleDefault.Foreground(thunder.Colors.Accent).Background(thunder.Colors.Background)).
		SetButtonActivatedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Accent))

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

	popup := NewDefaultPopup(modal, id, true, Center)

	modal.SetDoneFunc(func(indx int, label string) {
		thunder.Popups.Hide(popup)
		handler(indx, label)
	})

	thunder.Popups.Show(popup)
}

func errorPopup(err error) {
	timeoutPopup(err.Error(), 2*time.Second, tcell.ColorRed, tcell.ColorRed)
}

func infoPopup(text string) {
	timeoutPopup(text, 2*time.Second, thunder.Colors.Accent, thunder.Colors.Foreground)
}

func timeoutPopup(text string, timeout time.Duration, textColor, borderColor tcell.Color) {
	id := fmt.Sprintf("timeout_popup-%d", popupCount.Load())

	textView := tview.NewTextView().
		SetText(text).
		SetTextColor(textColor).SetTextAlign(tview.AlignCenter)

	textView.SetBackgroundColor(thunder.Colors.Background).SetBorder(true).
		SetBorderColor(borderColor).SetBorderPadding(1, 1, 1, 1)

	popup := NewTimeoutPopup(textView, id, false, TopRight, timeout)

	thunder.Popups.Show(popup)
}

func topRight(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 40, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(nil, 0, 10, false), width, 1, false).
		AddItem(nil, 0, 1, false)
}

func center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(nil, 0, 1, false), width, 1, false).
		AddItem(nil, 0, 1, false)
}
