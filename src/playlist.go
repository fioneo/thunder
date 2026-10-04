package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Playlist struct {
	*tview.Pages
	List        *tview.List
	songs       []Song
	updateTitle func()
	currentIdx  int
	currentSong int
	mode        PlayMode
	name        string
	path        string
	isModified  bool
}

func NewPlaylist(info PlaylistInfo) *Playlist {
	if info == nil {
		return &Playlist{}
	}
	list := tview.NewList()
	emptyPlaylistText := tview.NewTextView().SetText("empty playlist").
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background))
	pages := tview.NewPages().
		AddPage("playlist", list, true, true).
		AddPage("empty_playlist", emptyPlaylistText, true, false)

	playlist := &Playlist{
		Pages: pages,
		List:  list,
		mode:  DefaultMode,
		songs: info.Songs(),
		name:  info.Name(),
		path:  info.Path(),
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

	playlist.InsertAll()

	playlist.List.
		SetMainTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)).
		ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(tcell.ColorDefault)).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetBackgroundColor(thunder.Colors.Background)

	return playlist
}

func (p *Playlist) Insert(song Song) {
	p.List.InsertItem(len(p.songs), song.Name(), "", 0, nil)
	p.songs = append(p.songs, song)
	p.Update()
	p.SetIsModified(true)
}

func (p *Playlist) Remove(i int) {
	if len(p.songs) == 0 {
		return
	}

	if i < 0 || i >= len(p.songs) {
		return
	}

	p.List.RemoveItem(i)
	p.songs = append(p.songs[:i], p.songs[i+1:]...)

	if i == len(p.songs) {
		p.currentIdx--
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

func (p *Playlist) InsertAll() {
	for i, song := range p.songs {
		p.List.InsertItem(i, song.Name(), "", 0, nil)
	}
	p.CheckIsEmpty()
}

func (p *Playlist) CheckIsEmpty() {
	if len(p.songs) == 0 {
		p.SwitchToPage("empty_playlist")
	} else {
		p.SwitchToPage("playlist")
	}
}

func (p *Playlist) Update() {
	p.CheckIsEmpty()
	p.UpdateTitle()
}

func (p *Playlist) SetIsModified(modified bool) {
	p.isModified = modified
}
