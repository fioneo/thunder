package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"path/filepath"
)

type SelectionMode int

const (
	DefaultSelection SelectionMode = iota
	MultiplySelection
)

var allowedExt = ".mp3"

type Library struct {
	*List
	files       []File
	currentPath string
	currentIdx  int
	idxHistory  map[string]int
	mode        SelectionMode
	selected    []int
}

func NewLibrary(dir string) *Library {
	path := filepath.Join(dir, "Music/playlists")
	list := NewList()

	library := &Library{
		List:        list,
		currentPath: path,
		mode:        DefaultSelection,
		idxHistory:  make(map[string]int),
	}
	library.readDir(path)
	library.UpdateInfo()

	// library.SetSelectedFunc(func(idx int, _ string) {
	// 	library.currentIdx = idx
	// })

	library.SetChangedFunc(func(idx int, _ string) {
		library.currentIdx = idx
		library.UpdateInfo()
	})

	library.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {

		}
		switch e.Rune() {
		// case ' ':
		// 	return nil
		case 'l':
			// if library.mode == MultiplySelection {
			// 	if len(library.selected) == 0 {
			// 		return e
			// 	}
			// 	songs := filesToSongs(library.selected)
			// 	thunder.PlayPanel.playlist.MultiplyInsert(songs)
			// 	library.mode = DefaultSelection
			// 	return e
			// }
			if len(library.files) == 0 || library.currentIdx >= len(library.files) {
				return e
			}

			file := library.files[library.currentIdx]

			if !file.IsDir() {
				song := fileToSong(file)
				thunder.PlayPanel.playlist.Insert(song)
				return e
			}
			library.idxHistory[library.currentPath] = library.currentIdx
			library.Clear()
			library.readDir(file.Path())
		case 'h':
			path := filepath.Dir(library.currentPath)

			library.Clear()
			if err := library.readDir(path); err != nil {
				errorPopup(err)
				return e
			}
			if idx, ok := library.idxHistory[path]; ok {
				if idx < 0 || idx >= len(library.files) {
					return e
				}
				library.currentIdx = idx
				library.List.SetCurrentIdx(idx)
			}

			// 			if library.CheckIsModified() {
			// 				confirmationPopup("You have unsaved changes.\nAre you sure you want to close the current playlist?", func(_ int, label string) {
			// 					if label == "yes" {
			// 						playlistInfo := directoryToPlaylistInfo(library.currentDir, library.CurrentPath)
			//
			// 						thunder.PlayPanel.UpdateInfo(playlistInfo)
			// 					}
			// 				})
			// 			} else {
			// 				playlistInfo := directoryToPlaylistInfo(currentNode.Name(), currentNode.Path(), currentNode.Children())
			//
			// 				thunder.PlayPanel.UpdateInfo(playlistInfo)
			// 			}
			// 		case 'n':
			// 			if library.CheckIsModified() {
			// 				confirmationPopup("You have unsaved changes.\nAre you sure you want to create a new playlist?", func(_ int, label string) {
			// 					if label == "yes" {
			// 						playlistInfo := NewDefaultPlaylistInfo()
			//
			// 						thunder.PlayPanel.UpdateInfo(playlistInfo)
			// 					}
			// 				})
			// 			} else {
			// 				playlistInfo := NewDefaultPlaylistInfo()
			//
			// 				thunder.PlayPanel.UpdateInfo(playlistInfo)
			// 			}
		}
		return e
	})

	library.
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)).
		SetHighlightFullLine(true).
		SetSelectedFocusOnly(true).
		SetWrapAround(false).
		SetBorder(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(thunder.Colors.Accent).
		SetTitleColor(thunder.Colors.Accent).
		SetBorderPadding(0, 0, 1, 1).
		SetBackgroundColor(thunder.Colors.Background)

	return library
}

func (l *Library) readDir(path string) error {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	l.currentPath = path
	files := make([]File, 0, len(dirEntries))
	i := 0
	for _, entry := range dirEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) != allowedExt {
			continue
		}
		path := filepath.Join(path, entry.Name())
		name := entry.Name()

		file := NewFile(name, path, entry.IsDir())

		files = append(files, file)

		style := l.songStyle()
		selectedStyle := l.songSelectedStyle()

		if entry.IsDir() {
			style = l.playlistStyle()
			selectedStyle = l.playlistSelectedStyle()
		}

		l.List.InsertItem(i, name, &style, &selectedStyle)
		i++
	}
	l.files = files
	l.UpdateInfo()

	return nil
}

func (l *Library) Clear() {
	l.currentIdx = 0
	l.List.Clear()
	l.files = nil
}

func (l *Library) UpdateInfo() {
	idx := l.currentIdx
	if len(l.files) > 0 {
		idx++
	}
	title := fmt.Sprintf("─ %s ──┤ %d / %d ├",
		"Library",
		idx,
		len(l.files))
	l.SetTitle(title)
}

func (l *Library) CheckIsModified() bool {
	return thunder.PlayPanel.playlist.isModified
}

func (l *Library) songStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)
}

func (l *Library) songSelectedStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)
}

func (l *Library) playlistStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(thunder.Colors.Playlist).Background(thunder.Colors.Background)
}

func (l *Library) playlistSelectedStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Playlist)
}
