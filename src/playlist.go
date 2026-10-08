package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Playlist struct {
	*tview.Pages
	List        *List
	songs       []Song
	updateTitle func()
	currentIdx  int
	currentSong int
	mode        PlayMode
	name        string
	path        string
	isModified  bool
	unique      map[string]struct{}
}

func NewPlaylist(info PlaylistInfo) *Playlist {
	if info == nil {
		return &Playlist{}
	}
	list := NewList()
	emptyPlaylistText := tview.NewTextView().SetText("empty playlist").
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background))
	pages := tview.NewPages().
		AddPage("playlist", list, true, true).
		AddPage("empty_playlist", emptyPlaylistText, true, false)

	playlist := &Playlist{
		Pages:  pages,
		List:   list,
		mode:   DefaultMode,
		name:   info.Name(),
		path:   info.Path(),
		unique: make(map[string]struct{}, len(info.Songs())),
	}

	playlist.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		case 'r':
			idx := playlist.currentIdx
			if len(playlist.songs) == 0 ||
				idx < 0 ||
				idx >= len(playlist.songs) {
				return e
			}
			song := playlist.songs[idx]
			popupText := fmt.Sprintf("[ %s ]\nAre you sure you want to remove the current file from playlist?", song.Name())
			confirmationPopup(popupText, func(_ int, label string) {
				if label == "yes" {
					playlist.Remove(idx)
				}
			})
		}
		return e
	})

	playlist.List.SetChangedFunc(func(idx int, _ string) {
		playlist.currentIdx = idx
	})

	for _, song := range info.Songs() {
		playlist.Insert(song)
	}

	playlist.UpdateInfo()

	playlist.List.
		SetSelectedFocusOnly(true).
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetBackgroundColor(thunder.Colors.Background)
	return playlist
}

func (p *Playlist) Insert(song Song) {
	if _, ok := p.unique[song.Path()]; ok {
		popupText := fmt.Sprintf("[ %s ]\nThis song is already on the playlist.\nDo you want to add it again?", song.Name())
		confirmationPopup(popupText, func(_ int, label string) {
			if label == "no" || label == "" {
				return
			}
			p.List.InsertItem(len(p.songs), song.Name(), nil, nil)
			p.songs = append(p.songs, song)
			itemIdx := len(p.songs) - 1
			p.unique[song.Path()] = struct{}{}
			p.List.SetCurrentIdx(itemIdx)
			p.List.SetItemOffset(itemIdx)
			p.SetIsModified(true)
			p.UpdateInfo()
		})
	} else {
		p.List.InsertItem(len(p.songs), song.Name(), nil, nil)
		p.songs = append(p.songs, song)
		itemIdx := len(p.songs) - 1
		p.unique[song.Path()] = struct{}{}
		p.List.SetCurrentIdx(itemIdx)
		p.List.SetItemOffset(itemIdx)
		p.SetIsModified(true)
		p.UpdateInfo()
	}
}

func (p *Playlist) MultiplyInsert(songs []Song) {
	for _, song := range songs {
		p.Insert(song)
	}
}

func (p *Playlist) Remove(idx int) {
	if len(p.songs) == 0 {
		return
	}

	if idx < 0 || idx >= len(p.songs) {
		return
	}
	song := p.songs[idx]

	delete(p.unique, song.Path())
	p.List.RemoveItem(idx)
	p.songs = append(p.songs[:idx], p.songs[idx+1:]...)

	if p.currentIdx > idx || p.currentIdx == len(p.songs) {
		p.currentIdx--
	}

	p.UpdateInfo()

}

func (p *Playlist) RemoveAll(indexes []int) {
	for i, _ := range indexes {
		p.Remove(i)
	}
}

func (p *Playlist) TogglePause() {
	thunder.Player.TogglePause()
}

func (p *Playlist) SwitchMode() {
	switch p.mode {
	case DefaultMode:
		p.mode = RepeatMode
	case RepeatMode:
		p.mode = RandomMode
	case RandomMode:
		p.mode = DefaultMode
	}
}

func (p *Playlist) SetUpdateTitleFunc(deligate func()) {
	p.updateTitle = deligate
}

func (p *Playlist) UpdateTitle() {
	p.updateTitle()
}

func (p *Playlist) Get(i int) Song {
	if i < 0 || i >= len(p.songs) {
		return nil
	}

	return p.songs[i]
}

func (p *Playlist) Play(i int) {
	//TODO
}

func (p *Playlist) CheckIsEmpty() {
	if len(p.songs) == 0 {
		p.SwitchToPage("empty_playlist")
	} else {
		p.SwitchToPage("playlist")
	}
}

func (p *Playlist) UpdateInfo() {
	p.CheckIsEmpty()
	if p.updateTitle == nil {
		return
	}
	p.UpdateTitle()
}

func (p *Playlist) SetIsModified(modified bool) {
	p.isModified = modified
}
