package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Queue struct {
	*tview.Pages
	List        *List
	mode        PlayMode
	songs       []Song
	currentSong Song
	currentIdx  int
	done        chan struct{}
	updateTitle func()
}

func NewQueue() *Queue {
	list := NewList()
	emptyQueueText := tview.NewTextView().SetText("  empty queue.").
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background))
	pages := tview.NewPages().
		AddPage("queue", list, true, true).
		AddPage("empty_queue", emptyQueueText, true, false)

	queue := &Queue{
		Pages: pages,
		List:  list,
		mode:  DefaultMode,
		done:  make(chan struct{}, 1),
	}
	pages.SwitchToPage("empty_queue")

	queue.List.SetChangedFunc(func(idx int, _ string) {
		queue.currentIdx = idx
		queue.UpdateInfo()
	})

	queue.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			if len(queue.List.GetSelectedItems()) != 0 {
				queue.List.ClearSelected()
				queue.UpdateTitle()
			}
		}
		return e
	})

	queue.List.
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)).
		SetSelectedFocusOnly(true).
		SetCurrentItemStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Selected)).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetBackgroundColor(thunder.Colors.Background)
	return queue
}

func (q *Queue) Grow(capacity int) {
	q.songs = make([]Song, 0, capacity)
	q.List.Grow(capacity)
}

func (q *Queue) Length() int {
	return len(q.songs)
}

func (q *Queue) Remove(i int) {
	if len(q.songs) == 0 {
		return
	}

	if i < 0 || i >= len(q.songs) {
		return
	}

	q.List.RemoveItem(i)
	q.songs = append(q.songs[:i], q.songs[i+1:]...)

	if i == len(q.songs) {
		q.currentIdx--
	}
}

func (q *Queue) next() {
	if q.Length() == 0 {
		return
	}

	song := q.Peek()

	// play current
	q.Play(song)
}

func (q *Queue) Clear() {
	q.currentIdx = 0
	q.currentSong = nil
	q.songs = nil
	q.List.Clear()
}

func (q *Queue) ChangeMode() {
	// TODO
}

func (q *Queue) Peek() Song {
	return q.songs[0]
}

func (q *Queue) GetCurrentSong() Song {
	return q.currentSong
}

func (q *Queue) enqueue(song Song) {
	q.songs = append(q.songs, song)
	q.List.InsertItem(q.Length(), song, song.Name(), nil, nil)

}

func (q *Queue) dequeue() Song {
	song := q.songs[0]
	q.songs = q.songs[1:]
	q.List.RemoveItem(0)

	return song
}

func (q *Queue) SetUpdateTitleFunc(updateTitleFunc func()) {
	q.updateTitle = updateTitleFunc
}

func (q *Queue) UpdateTitle() {
	q.updateTitle()
}

func (q *Queue) CheckIsEmpty() {
	if q.Length() == 0 {
		q.SwitchToPage("empty_queue")
	} else {
		q.SwitchToPage("queue")
	}
}

func (q *Queue) UpdateInfo() {
	q.CheckIsEmpty()
	if q.updateTitle == nil {
		return
	}
	q.UpdateTitle()
}

func (q *Queue) PlayQueue(songs []Song) {
	q.Clear()

	if len(songs) == 0 {
		return
	}

	for _, song := range songs {
		q.enqueue(song)
	}

	q.next()
}

func (q *Queue) Play(song Song) {
	q.currentSong = song

	style, selectedStyle := q.currentSongStyle()
	q.List.SetItemHighlightedStyle(0, style, selectedStyle)

	thunder.Player.SetSongFinishFunc(func(song Song) {
		q.dequeue()
	})

	if err := thunder.Player.Play(song); err != nil {
		errorPopup(err)
	}
}

func (q *Queue) currentSongStyle() (*tcell.Style, *tcell.Style) {
	style := tcell.StyleDefault.Foreground(thunder.Colors.Highlighted).Background(thunder.Colors.Background)
	selectedStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Highlighted)

	return &style, &selectedStyle
}

func (q *Queue) IsRunning() bool {
	return thunder.Player.IsRunning()
}
