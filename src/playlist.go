package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	playlistPageList  = "list"
	playlistPageEmpty = "empty"
)

type Playlist struct {
	*tview.Pages
	list *List
	ctrl *PlaybackController

	songs    []Song
	name     string
	path     string
	modified bool
	onUpdate func()
}

func NewPlaylist(info PlaylistInfo, ctrl *PlaybackController) *Playlist {
	colors := thunder.Colors

	list := NewList()
	list.ApplySongStyles(colors).SetBackgroundColor(colors.Background)

	empty := tview.NewTextView().
		SetText(" empty playlist").
		SetTextStyle(tcell.StyleDefault.Foreground(colors.Song).Background(colors.Background))

	p := &Playlist{
		Pages: tview.NewPages().
			AddPage(playlistPageList, list, true, false).
			AddPage(playlistPageEmpty, empty, true, true),
		list: list,
		ctrl: ctrl,
	}

	list.SetChangedFunc(func(int, string) { p.notify() })
	p.SetInputCapture(p.handleKey)
	p.SetInfo(info)
	return p
}

func (p *Playlist) SetInfo(info PlaylistInfo) {
	p.songs = nil
	p.name, p.path = "", ""
	if info != nil {
		p.name, p.path = info.Name(), info.Path()
		p.songs = append(p.songs, info.Songs()...)
	}
	p.modified = false
	p.list.ClearSelected()
	p.list.SetCurrentIdx(0)
	p.refresh(0)
}

func (p *Playlist) SetUpdateFunc(fn func()) { p.onUpdate = fn }

func (p *Playlist) notify() {
	if p.onUpdate != nil {
		p.onUpdate()
	}
}

func (p *Playlist) Title() string {
	pos := 0
	if len(p.songs) > 0 {
		pos = p.list.GetCurrentIdx() + 1
	}
	return fmt.Sprintf("─ %s ──┤ %d / %d | %s ├", p.name, pos, len(p.songs), p.ctrl.Mode().ToString())
}

func (p *Playlist) GetModified() bool { return p.modified }

func (p *Playlist) refresh(cursor int) {
	entries := make([]ListEntry, len(p.songs))
	for i, song := range p.songs {
		entries[i] = ListEntry{Item: song, Text: song.Name()}
	}
	p.list.Reset(entries)
	p.list.SetCurrentIdx(min(max(cursor, 0), len(p.songs)-1))

	if len(p.songs) == 0 {
		p.SwitchToPage(playlistPageEmpty)
	} else {
		p.SwitchToPage(playlistPageList)
	}
	p.notify()
}

func (p *Playlist) add(songs []Song) (added, skipped int) {
	seen := make(map[string]struct{}, len(p.songs)+len(songs))
	for _, s := range p.songs {
		seen[s.Path()] = struct{}{}
	}

	for _, s := range songs {
		if _, dup := seen[s.Path()]; dup {
			skipped++
			continue
		}
		seen[s.Path()] = struct{}{}
		p.songs = append(p.songs, s)
		added++
	}

	if added > 0 {
		p.modified = true
		p.refresh(len(p.songs) - 1)
	}
	return added, skipped
}

func (p *Playlist) Insert(song Song) {
	if _, skipped := p.add([]Song{song}); skipped > 0 {
		infoPopup(fmt.Sprintf("[ %s ]\nThis song is already on the playlist.", song.Name()))
	}
}

func (p *Playlist) InsertAll(songs []Song) {
	added, skipped := p.add(songs)
	if skipped > 0 {
		infoPopup(fmt.Sprintf("Songs added: %d\nSongs already in the playlist: %d", added, skipped))
	}
}

func (p *Playlist) remove(paths ...string) {
	drop := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		drop[path] = struct{}{}
	}

	kept := p.songs[:0]
	for _, s := range p.songs {
		if _, ok := drop[s.Path()]; !ok {
			kept = append(kept, s)
		}
	}
	clear(p.songs[len(kept):])
	if len(kept) == len(p.songs) {
		return
	}

	p.songs = kept
	p.modified = true
	p.refresh(p.list.GetCurrentIdx())
}

func (p *Playlist) targetSongs(idx int) []Song {
	if selected := p.list.GetSelectedItems(); len(selected) > 0 {
		songs := make([]Song, len(selected))
		for i, item := range selected {
			songs[i] = item.(Song)
		}
		return songs
	}
	if idx >= 0 && idx < len(p.songs) {
		return []Song{p.songs[idx]}
	}
	return nil
}

func (p *Playlist) handleKey(e *tcell.EventKey) *tcell.EventKey {
	idx := p.list.GetCurrentIdx()

	if e.Key() == tcell.KeyEscape {
		p.list.ClearSelected()
		return e
	}

	switch e.Rune() {
	case ' ':
		if idx >= 0 && idx < len(p.songs) && len(p.list.GetSelectedItems()) == 0 {
			p.ctrl.PlayFrom(p.songs[idx:])
		}
	case 'a':
		p.addToQueue(p.targetSongs(idx))
	case 'd':
		p.confirmRemove(idx)
	}
	return e
}

func (p *Playlist) addToQueue(songs []Song) {
	if len(songs) == 0 {
		return
	}

	added, skipped := p.ctrl.Enqueue(songs...)
	p.list.ClearSelected()

	switch {
	case len(songs) > 1:
		infoPopup(fmt.Sprintf("Songs added to the queue: %d\nAlready in the queue: %d", added, skipped))
	case added == 1:
		infoPopup(fmt.Sprintf("[ %s ]\nSong added to the queue", songs[0].Name()))
	default:
		infoPopup(fmt.Sprintf("[ %s ]\nThis song is already in the queue", songs[0].Name()))
	}
}

func (p *Playlist) confirmRemove(idx int) {
	songs := p.targetSongs(idx)
	if len(songs) == 0 {
		return
	}

	paths := make([]string, len(songs))
	for i, s := range songs {
		paths[i] = s.Path()
	}

	text := fmt.Sprintf("[ %s ]\nAre you sure you want to remove the current song from playlist?", songs[0].Name())
	if len(songs) > 1 {
		text = fmt.Sprintf("Are you sure you want to remove the selected songs from the playlist?\nNumber of songs to be removed: %d", len(songs))
	}

	confirmationPopup(text, func(_ int, label string) {
		if label == "yes" {
			p.remove(paths...)
			p.list.ClearSelected()
		}
	})
}
