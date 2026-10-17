package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	playlistPage = "playlist"
	queuePage    = "queue"
)

type PlayPanel struct {
	*tview.Pages
	playlist *Playlist
	queue    *QueueView

	current string
}

func NewPlayPanel(info PlaylistInfo, ctrl *PlaybackController) *PlayPanel {
	playlist := NewPlaylist(info, ctrl)
	queue := NewQueueView(ctrl)

	p := &PlayPanel{
		Pages: tview.NewPages().
			AddPage(playlistPage, playlist, true, true).
			AddPage(queuePage, queue, true, false),
		playlist: playlist,
		queue:    queue,
		current:  playlistPage,
	}

	playlist.SetUpdateFunc(p.refreshTitle)
	queue.SetUpdateFunc(p.refreshTitle)

	p.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Rune() == 'q' {
			p.togglePage()
		}
		return e
	})

	p.SetBorder(true).
		SetBorderColor(thunder.Colors.Foreground).
		SetTitleColor(thunder.Colors.Foreground).
		SetBackgroundColor(thunder.Colors.Background)

	p.queue.Render(ctrl.Snapshot())
	p.refreshTitle()
	return p
}

func (p *PlayPanel) togglePage() {
	if p.current == playlistPage {
		p.current = queuePage
	} else {
		p.current = playlistPage
	}
	p.SwitchToPage(p.current)

	if p.HasFocus() && thunder != nil && thunder.App != nil {
		thunder.App.SetFocus(p)
	}
	p.refreshTitle()
}

func (p *PlayPanel) refreshTitle() {
	if p.current == queuePage {
		p.SetTitle(p.queue.Title())
		return
	}
	p.SetTitle(p.playlist.Title())
}

func (p *PlayPanel) Render(s Snapshot) {
	p.queue.Render(s)
	p.refreshTitle()
}

func (p *PlayPanel) UpdateInfo(info PlaylistInfo) {
	p.playlist.SetInfo(info)
}
