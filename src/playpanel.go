package main

import (
	"github.com/rivo/tview"
)

type PlayPanel struct {
	*tview.Pages
	playlist *Playlist
	queue    *Queue
}

func NewPlayPanel(info PlaylistInfo) *PlayPanel {
	if info == nil {
		return &PlayPanel{}
	}
	playlist := NewPlaylist(info)
	queue := NewQueue(info.Songs())

	pages := tview.NewPages().
		AddPage("playlist", playlist, true, true).
		AddPage("queue", queue, true, false)

	playpanel := &PlayPanel{
		Pages:    pages,
		playlist: playlist,
		queue:    queue,
	}

	return playpanel
}
