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
	*tview.Flex
	nowPlaying *tview.TextView
	pages      *tview.Pages
	list       *List

	ctrl     *PlaybackController
	snap     Snapshot
	onUpdate func()
}

func NewQueueView(ctrl *PlaybackController) *QueueView {
	colors := thunder.Colors
	background := tcell.StyleDefault.Background(colors.Background)

	list := NewList()
	list.ApplySongStyles(colors).SetBackgroundColor(colors.Background)

	empty := tview.NewTextView().
		SetText("  queue is empty").
		SetTextStyle(background.Foreground(colors.Song))

	nowPlaying := tview.NewTextView().
		SetTextStyle(background.Foreground(colors.Highlighted))

	pages := tview.NewPages().
		AddPage(queuePageList, list, true, false).
		AddPage(queuePageEmpty, empty, true, true)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nowPlaying, 1, 0, false).
		AddItem(pages, 0, 1, true)

	q := &QueueView{
		Flex:       flex,
		nowPlaying: nowPlaying,
		pages:      pages,
		list:       list,
		ctrl:       ctrl,
	}

	list.SetChangedFunc(func(int, string) { q.notify() })
	q.SetInputCapture(q.handleKey)
	return q
}

func (q *QueueView) SetUpdateFunc(fn func()) { q.onUpdate = fn }

func (q *QueueView) notify() {
	if q.onUpdate != nil {
		q.onUpdate()
	}
}

func (q *QueueView) Render(s Snapshot) {
	if s.Version < q.snap.Version {
		return
	}
	q.snap = s
	entries := make([]ListEntry, len(s.Upcoming))
	for i, song := range s.Upcoming {
		entries[i] = ListEntry{Item: song, Text: song.Name()}
	}
	q.list.Reset(entries)

	if len(entries) == 0 {
		q.pages.SwitchToPage(queuePageEmpty)
	} else {
		q.pages.SwitchToPage(queuePageList)
	}

	q.nowPlaying.SetText(nowPlayingText(s))
	q.notify()
}

func nowPlayingText(s Snapshot) string {
	if s.Now == nil {
		return "  nothing playing"
	}

	return "  " + s.Now.Name()
}

func (q *QueueView) Title() string {
	return fmt.Sprintf("─ Queue ──┤ %d %s │ %s ├",
		len(q.snap.Upcoming), songWord(len(q.snap.Upcoming)), q.snap.Mode.ToString())
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
