package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Queue struct {
	*tview.Pages
	List        *tview.List
	updateTitle func()
	mode        PlayMode
	songs       []Song
	currentIdx  int
	currentSong int
}

func NewQueue(songs []Song) *Queue {
	list := tview.NewList()
	emptyQueueText := tview.NewTextView().SetText("empty queue").
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background))
	pages := tview.NewPages().
		AddPage("playlist", list, true, true).
		AddPage("empty_queue", emptyQueueText, true, false)

	queue := &Queue{
		Pages: pages,
		List:  list,
		mode:  DefaultMode,
		songs: songs,
	}
	if len(songs) == 0 {
		pages.SwitchToPage("empty_queue")
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		}
		return nil
	})

	queue.InsertAll()

	queue.List.
		SetMainTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)).
		ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(tcell.ColorDefault)).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetBackgroundColor(thunder.Colors.Background)

	return queue
}

func (q *Queue) Length() int {
	return len(q.songs)
}

func (q *Queue) Insert(song Song) {
	q.List.InsertItem(len(q.songs), song.Name(), "", 0, nil)
	q.songs = append(q.songs, song)
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

func (q *Queue) Get(i int) Song {
	if i < 0 || i >= len(q.songs) {
		return nil
	}
	return q.songs[i]
}

func (q *Queue) next() int {
	q.currentSong = q.currentSong + 1
	if q.currentSong >= len(q.songs) {
		q.currentSong = 0
	}
	return q.currentSong
}

func (q *Queue) Clear() {
	q.songs = []Song{}
	q.List.Clear()
}

func (q *Queue) ChangeMode() {
	// TODO
}

func (q *Queue) first() Song {
	first := q.songs[0]
	q.songs = q.songs[1:]
	return first
}

func (q *Queue) dequeue() Song {
	if len(q.songs) == 0 {
		return nil
	}

	song := q.first()

	return song
}

func (q *Queue) SetUpdateTitleFunc(deligate func()) {
	q.updateTitle = deligate
}

func (q *Queue) UpdateTitle() {
	q.updateTitle()
}

func (q *Queue) Play() {
	// TODO
}

func (q *Queue) InsertAll() {
	for i, song := range q.songs {
		q.List.InsertItem(i, song.Name(), "", 0, nil)
	}
}
