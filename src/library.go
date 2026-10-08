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
	*tview.Frame
	List        *List
	files       []File
	currentPath string
	currentDir  string
	currentIdx  int
	idxHistory  map[string]int
	mode        SelectionMode
	selected    []int
}

func NewLibrary(dir string) *Library {
	path := filepath.Join(dir, "Music/playlists")

	list := NewList()
	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)

	library := &Library{
		Frame:       frame,
		List:        list,
		currentPath: path,
		mode:        DefaultSelection,
		idxHistory:  make(map[string]int),
	}

	if err := library.readDir(path); err != nil {
		exit(err)
	}

	library.List.SetChangedFunc(func(idx int, _ string) {
		library.currentIdx = idx
		library.UpdateInfo()
	})

	library.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
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
			if len(library.files) == 0 ||
				library.currentIdx < 0 ||
				library.currentIdx >= len(library.files) {
				return e
			}

			file := library.files[library.currentIdx]

			if !file.IsDir() {
				song := fileToSong(file)
				thunder.PlayPanel.playlist.Insert(song)
				return e
			}
			currentIdx := library.currentIdx
			currentPath := library.currentPath
			path := file.Path()

			if err := library.readDir(path); err != nil {
				errorPopup(err)
				return e
			}
			library.idxHistory[currentPath] = currentIdx
		case 'h':
			path := filepath.Dir(library.currentPath)

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
		case 'd':
			idx := library.currentIdx
			if len(library.files) == 0 ||
				idx < 0 ||
				idx >= len(library.files) {
				return e
			}
			file := library.files[idx]
			popupText := fmt.Sprintf("[ %s ]\nAre you sure you want to delete the current file?", file.Name())
			confirmationPopup(popupText, func(_ int, label string) {
				if label == "yes" {
					if err := library.Remove(idx); err != nil {
						errorPopup(err)
					}
				}
			})

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

	library.List.
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)).
		SetHighlightFullLine(true).
		SetSelectedFocusOnly(true).
		SetWrapAround(false).
		SetBackgroundColor(thunder.Colors.Background)

	library.
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
	files := make([]File, 0, len(dirEntries))
	l.Clear()
	idx := 0
	for _, entry := range dirEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) != allowedExt {
			continue
		}
		path := filepath.Join(path, entry.Name())
		name := entry.Name()

		file := NewFile(name, path, entry.IsDir())

		files = append(files, file)

		style, selectedStyle := l.GetStyle(file)
		l.List.InsertItem(idx, file.Name(), style, selectedStyle)
		idx++
	}

	l.files = files
	l.UpdateInfo()
	l.currentPath = path
	l.currentDir = filepath.Base(path)
	l.Frame.Clear()
	dir := l.currentDir
	if l.currentDir != "/" {
		dir += "/"
	}
	l.Frame.AddText(dir, true, tview.AlignLeft, tcell.ColorGreen)

	return nil
}

func (l *Library) Remove(idx int) error {
	file := l.files[idx]
	if err := os.Remove(file.Path()); err != nil {
		return err
	}

	l.List.RemoveItem(idx)
	l.files = append(l.files[:idx], l.files[idx+1:]...)

	if len(l.files) == 0 {
		return nil
	}

	if l.currentIdx > idx || l.currentIdx == len(l.files) {
		l.currentIdx--
	}

	l.UpdateInfo()

	return nil
}

// func (l *Library) RemoveAll() {}

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

	// l.CheckIsEmpty()
}

func (l *Library) CheckIsModified() bool {
	return thunder.PlayPanel.playlist.isModified
}

// func (l *Library) CheckIsEmpty() {
// 	if len(l.files) == 0 {
// 		l.SwitchToPage("empty_playlist")
// 	} else {
// 		l.SwitchToPage("playlist")
// 	}
// }

func (l *Library) GetStyle(file File) (*tcell.Style, *tcell.Style) {
	style := l.songStyle()
	selectedStyle := l.songSelectedStyle()

	if file.IsDir() {
		style = l.playlistStyle()
		selectedStyle = l.playlistSelectedStyle()
	}

	return &style, &selectedStyle
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
