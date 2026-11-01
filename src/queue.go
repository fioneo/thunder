package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	queuePageList  = "list"
	queuePageEmpty = "empty"
)

type QueueView struct {
	*tview.Pages
	frame *tview.Frame
	list  *List

	ctrl     *PlaybackController
	snap     Snapshot
	onUpdate func()
}

func NewQueueView(ctrl *PlaybackController) *QueueView {
	colors := colors

	list := NewList()
	list.ApplySongStyles().SetBackgroundColor(colors.Background)

	empty := tview.NewTextView().SetText("  Empty queue").
		SetTextStyle(tcell.StyleDefault.Background(colors.Background).Foreground(colors.Song))

	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)

	pages := tview.NewPages().
		AddPage(queuePageList, frame, true, true).
		AddPage(queuePageEmpty, empty, true, false)

	q := &QueueView{
		Pages: pages,
		frame: frame,
		list:  list,
		ctrl:  ctrl,
	}

	q.Primitive = q

	list.SetChangedFunc(func(int, string) { q.UpdateInfo() })

	q.frame.SetBackgroundColor(colors.Background)
	q.SetInputCapture(q.handleKey)

	q.SetBorder(true).
		SetBorderColor(colors.Foreground).
		SetTitleColor(colors.Foreground).
		SetBackgroundColor(colors.Background)

	return q
}

func (q *QueueView) UpdateInfo() {
	q.SetTitle(fmt.Sprintf("─ Queue ──┤ %d %s │ %s ├",
		len(q.snap.Upcoming), songWord(len(q.snap.Upcoming)), q.snap.Mode.ToString()))
}

func (q *QueueView) Render(s Snapshot) {
	if s.Version < q.snap.Version {
		return
	}
	q.frame.Clear()
	q.snap = s
	entries := make([]ListEntry, len(s.Upcoming))
	for i, song := range s.Upcoming {
		entries[i] = ListEntry{Item: song, Text: song.Name()}
	}
	q.list.Reset(entries)

	if q.snap.Now != nil {
		q.frame.AddText("  "+q.snap.Now.Name(), true, tview.AlignLeft, colors.Highlighted)
	}

	if len(entries) == 0 && q.snap.Now == nil {
		q.SwitchToPage(queuePageEmpty)
	} else {
		q.SwitchToPage(queuePageList)
	}

	q.UpdateInfo()
}

func (q *QueueView) refresh() {
	q.Render(q.ctrl.Snapshot())
}

func (q *QueueView) selectedPaths() []string {
	selected := q.list.GetSelectedItems()
	paths := make([]string, len(selected))
	for i, item := range selected {
		paths[i] = item.Path()
	}
	return paths
}

func (q *QueueView) handleKey(e *tcell.EventKey) *tcell.EventKey {
	idx := q.list.GetCurrentIdx()
	hasCurrent := idx >= 0 && idx < len(q.snap.Upcoming)

	switch e.Key() {
	case tcell.KeyEscape:
		q.list.ClearSelected()
		return e

	case tcell.KeyUp, tcell.KeyDown:
		if e.Modifiers()&tcell.ModCtrl == 0 {
			return e
		}
		if hasCurrent && len(q.list.GetSelectedItems()) == 0 {
			delta := 1
			if e.Key() == tcell.KeyUp {
				delta = -1
			}
			q.list.SetCurrentIdx(q.ctrl.Move(idx, delta))
			q.refresh()
		}
		return nil
	}

	switch e.Rune() {
	case ' ':
		if hasCurrent {
			q.ctrl.PlayAt(idx)
			q.refresh()
		}
	case 'd':
		q.confirmRemove(idx, hasCurrent)
	case 'c':
		if len(q.snap.Upcoming) > 0 {
			confirmationPopup("Are you sure you want to clear the queue?", func(_ int, label string) {
				if label == "yes" {
					q.ctrl.Clear()
					q.refresh()
				}
			})
		}
	case 'h':
		q.ctrl.Shuffle()
	}
	return e
}

func (q *QueueView) confirmRemove(idx int, hasCurrent bool) {
	paths := q.selectedPaths()
	text := ""

	switch {
	case len(paths) > 0:
		text = fmt.Sprintf("Are you sure you want to remove the selected songs from the queue?\nNumber of songs to be removed: %d", len(paths))
	case hasCurrent:
		song := q.snap.Upcoming[idx]
		paths = []string{song.Path()}
		text = fmt.Sprintf("[ %s ]\nAre you sure you want to remove the current song from queue?", song.Name())
	default:
		return
	}

	confirmationPopup(text, func(_ int, label string) {
		if label == "yes" {
			q.ctrl.Remove(paths...)
			q.list.ClearSelected()
			q.refresh()
		}
	})
}
