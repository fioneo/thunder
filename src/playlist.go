package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sort"
	"time"
)

type Playlist struct {
	*tview.Pages
	List        *List
	songs       []Song
	currentIdx  int
	currentSong int
	mode        PlayMode
	name        string
	path        string
	isModified  bool
	unique      map[string]struct{}
	updateTitle func()
}

func NewPlaylist(info PlaylistInfo) *Playlist {
	if info == nil {
		return &Playlist{}
	}
	list := NewList()
	emptyPlaylistText := tview.NewTextView().SetText(" empty playlist").
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
		idx := playlist.currentIdx

		if e.Key() == tcell.KeyEscape {
			if len(playlist.List.GetSelectedItems()) != 0 {
				playlist.List.ClearSelected()
				playlist.UpdateTitle()
			}
		}
		switch e.Rune() {
		case 'd':
			if len(playlist.songs) == 0 ||
				idx < 0 ||
				idx >= len(playlist.songs) {
				return e
			}
			selectedItems := playlist.List.GetSelectedItems()
			if len(selectedItems) > 0 {
				popupText := fmt.Sprintf("Are you sure you want to remove the selected songs from playlist?\nNumber of songs to be removed: %d", len(selectedItems))
				confirmationPopup(popupText, func(_ int, label string) {
					if label == "yes" {
						playlist.RemoveSelected()
						playlist.List.ClearSelected()
						playlist.UpdateInfo()
					}
				})
				return e
			}

			song := playlist.songs[idx]
			popupText := fmt.Sprintf("[ %s ]\nAre you sure you want to remove the current song from playlist?", song.Name())
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
		playlist.UpdateInfo()
	})

	for i, song := range info.Songs() {
		playlist.List.InsertItem(i, song, song.Name(), nil, nil)
		playlist.songs = append(playlist.songs, song)
		playlist.unique[song.Path()] = struct{}{}
	}

	playlist.UpdateInfo()

	playlist.List.
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)).
		SetSelectedFocusOnly(true).
		SetCurrentItemStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGreen)).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetBackgroundColor(thunder.Colors.Background)
	return playlist
}

func (p *Playlist) Insert(song Song) {
	if p.IsUnique(song) {
		p.add(song)
		return
	}
	text := fmt.Sprintf("[ %s ]\nThis song is already on the playlist.", song.Name())

	timeoutPopup(text, 3*time.Second)
}

func (p *Playlist) add(song Song) {
	itemIdx := len(p.songs)

	p.List.InsertItem(itemIdx, song, song.Name(), nil, nil)
	p.songs = append(p.songs, song)
	p.unique[song.Path()] = struct{}{}

	p.List.SetCurrentIdx(itemIdx)
	p.List.SetItemOffset(itemIdx)
	p.SetIsModified(true)
	p.UpdateInfo()
}

func (p *Playlist) InsertAll(songs []Song) {
	added := 0
	existing := 0
	for _, song := range songs {
		if p.IsUnique(song) {
			p.add(song)
			added++
		} else {
			existing++
		}
	}

	if existing > 0 {
		text := fmt.Sprintf("Songs added: %d\nSongs already in the playlist: %d", added, existing)
		timeoutPopup(text, 3*time.Second)
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

func (p *Playlist) RemoveSelected() {
	selectedItems := p.List.GetSelectedItems()

	if len(selectedItems) == 0 {
		return
	}

	indices := make([]int, 0, len(selectedItems))

	for _, item := range selectedItems {
		for i, s := range p.songs {
			if s.Path() == item.Path() {
				indices = append(indices, i)
				break
			}
		}
	}

	// This is crucial, because removing in ascending order shifts elements, and eventually we can get an index out of range (panic)
	sort.Slice(indices, func(i, j int) bool {
		return indices[i] > indices[j]
	})

	for _, idx := range indices {
		p.Remove(idx)

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

func (p *Playlist) SetUpdateTitleFunc(updateTitleFunc func()) {
	p.updateTitle = updateTitleFunc
}

func (p *Playlist) UpdateTitle() {
	p.updateTitle()
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

func (p *Playlist) IsUnique(song Song) bool {
	_, ok := p.unique[song.Path()]
	return !ok
	// IsUnique() = true = not in playlist ; IsUnique() = false = already in playlist
}
