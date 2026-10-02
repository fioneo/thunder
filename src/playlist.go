package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Playlist struct {
	*tview.List
	songs       []Song
	currentIdx  int
	currentSong int
	mode        PlayMode
	name        string
	path        string
}

func NewPlaylist(info PlaylistInfo) *Playlist {
	if info == nil {
		return &Playlist{}
	}
	list := tview.NewList().
		SetMainTextColor(tcell.ColorPurple).
		ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorPurple))


	playlist := &Playlist{
		List:  list,
		mode:  DefaultMode,
		songs: info.Songs(),
		name:  info.Name(),
		path:  info.Path(),
	}

	playlist.InsertAll()
	playlist.UpdateTitle()

	playlist.SetBorder(true).
		SetBorderColor(tcell.ColorBlue).
		SetTitleColor(tcell.ColorBlue).
		SetBorderPadding(0, 0, 1, 1)

	return playlist
}

func (p *Playlist) Insert(song Song) {
	p.List.InsertItem(len(p.songs), song.Name(), "", 0, nil)
	p.songs = append(p.songs, song)
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

func (p *Playlist) ChangeMode() {}

func (p *Playlist) UpdateTitle() {
	title := fmt.Sprintf(
		"─ %s ──┤ %d songs | %s ├",
		p.name,
		len(p.songs),
		p.mode.ToString(),
	)
	p.SetTitle(title)
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
}