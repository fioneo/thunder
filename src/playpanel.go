package main

import (
	"fmt"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
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


	pages.SetInputCapture((func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				pages.SwitchToPage("playlist")
			}
			switch event.Rune() {
				case 'q':
						currentPage,_ := pages.GetFrontPage()
						
						if currentPage == "playlist" {
								pages.SwitchToPage("queue")
								playpanel.queue.UpdateTitle()
								break	
						}
						pages.SwitchToPage("playlist")
						playpanel.playlist.UpdateTitle()
			}
			return event
					}))

	playlist.SetUpdateTitleFunc(func() {
				title := fmt.Sprintf(
					"─ %s ──┤ %d songs | %s ├",
					playpanel.playlist.name,
					len(playpanel.playlist.songs),
					playpanel.playlist.mode.ToString(),
				)
				playpanel.SetTitle(title)
			})
	

	queue.SetUpdateTitleFunc(func() {
				title := fmt.Sprintf(
					"─ %s ──┤ %d songs | %s ├",
					"Queue",
					len(playpanel.queue.songs),
					playpanel.queue.mode.ToString(),
				)
				playpanel.SetTitle(title)
			})

	playpanel.SetBlurFunc(func(){
		currentPage,_ := pages.GetFrontPage()
								
		if currentPage == "playlist" {
				playpanel.playlist.
				SetSelectedStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(tcell.ColorDefault))
		} else {
			playpanel.queue.
			SetSelectedStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(tcell.ColorDefault))
		}
	})

	playpanel.SetFocusFunc(func() {
		currentPage,_ := pages.GetFrontPage()
									
			if currentPage == "playlist" {
					playpanel.playlist.
					SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song))
			} else {
				playpanel.queue.
				SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song))
			}
	})
	
	playpanel.SetBorder(true).
		SetBorderColor(thunder.Colors.Foreground).
		SetTitleColor(thunder.Colors.Foreground).
		SetBorderPadding(0, 0, 1, 1).
		SetBackgroundColor(thunder.Colors.Background)

	playpanel.playlist.UpdateTitle()
	
	return playpanel
}

