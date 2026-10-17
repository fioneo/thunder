package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Library struct {
	*tview.Frame
	list        *List
	files       []File
	currentPath string
	currentIdx  int
	idxHistory  map[string]int
}

func NewLibrary(path string) *Library {
	list := NewList()
	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)

	library := &Library{
		Frame:       frame,
		list:        list,
		currentPath: path,
		idxHistory:  make(map[string]int),
	}

	if err := library.readDir(path); err != nil {
		exit(err)
	}

	library.list.SetChangedFunc(func(idx int, _ string) {
		library.currentIdx = idx
		library.UpdateInfo()
	})

	library.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		idx := library.currentIdx

		switch e.Key() {
		case tcell.KeyCtrlR:
			library.readDir(library.currentPath)
		case tcell.KeyEscape:
			if len(library.list.GetSelectedItems()) != 0 {
				library.list.ClearSelected()
				library.UpdateInfo()
			}
		}
		switch e.Rune() {
		case 'l':
			selectedItems := library.list.GetSelectedItems()

			if len(selectedItems) > 0 {
				files := make([]File, 0, len(selectedItems))

				for _, item := range selectedItems {
					files = append(files, item.(File))
				}

				songs := filesToSongs(files)
				thunder.PlayPanel.playlist.InsertAll(songs)
				library.list.ClearSelected()
				library.UpdateInfo()
				return e
			}

			if len(library.files) == 0 ||
				idx < 0 ||
				idx >= len(library.files) {
				return e
			}

			file := library.files[idx]

			if !file.IsDir() {
				song := fileToSong(file)
				thunder.PlayPanel.playlist.Insert(song)
				return e
			}

			currentPath := library.currentPath
			path := file.Path()

			if err := library.readDir(path); err != nil {
				errorPopup(err)
				return e
			}
			library.idxHistory[currentPath] = idx
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
				library.list.SetCurrentIdx(idx)
			}
		case 'd':
			selectedItems := library.list.GetSelectedItems()
			if len(selectedItems) > 0 {
				popupText := fmt.Sprintf("Are you sure you want to delete the selected files?\nNumber of files to be deleted: %d", len(selectedItems))
				confirmationPopup(popupText, func(_ int, label string) {
					if label == "yes" {
						library.RemoveSelected()
						library.list.ClearSelected()
						library.UpdateInfo()
					}
				})
				return e
			}

			if len(library.files) == 0 ||
				idx < 0 ||
				idx >= len(library.files) {
				return e
			}

			file := library.files[idx]

			if file.IsDir() {
				text := fmt.Sprintf("[ %s ]\nAre you sure you want to delete the current folder?\nThis will delete all files in it.", file.Name())
				confirmationPopup(text, func(_ int, label string) {
					if label == "yes" {
						if err := library.RemovePlaylist(idx); err != nil {
							errorPopup(err)
						}
					}
				})
				return e
			}

			text := fmt.Sprintf("[ %s ]\nAre you sure you want to delete the current file?", file.Name())

			confirmationPopup(text, func(_ int, label string) {
				if label == "yes" {
					if err := library.Remove(idx); err != nil {
						errorPopup(err)
					}
				}
			})

			library.UpdateInfo()
		case 'o':
			if library.CheckIsModified() {
				confirmationPopup("You have unsaved changes.\nAre you sure you want to close the current playlist?", func(_ int, label string) {
					if label == "yes" {
						playlistInfo := directoryToPlaylistInfo(filepath.Base(library.currentPath), library.currentPath, library.files)

						thunder.PlayPanel.UpdateInfo(playlistInfo)
					}
				})
			} else {
				playlistInfo := directoryToPlaylistInfo(filepath.Base(library.currentPath), library.currentPath, library.files)

				thunder.PlayPanel.UpdateInfo(playlistInfo)
			}
		case 'n':
			if library.CheckIsModified() {
				confirmationPopup("You have unsaved changes.\nAre you sure you want to create a new playlist?", func(_ int, label string) {
					if label == "yes" {
						playlistInfo := NewDefaultPlaylistInfo(library.currentPath)

						thunder.PlayPanel.UpdateInfo(playlistInfo)
					}
				})
			} else {
				playlistInfo := NewDefaultPlaylistInfo(library.currentPath)

				thunder.PlayPanel.UpdateInfo(playlistInfo)
			}
		}
		return e
	})

	colors := thunder.Colors
	library.list.ApplySongStyles(colors).SetBackgroundColor(colors.Background)

	library.
		SetBorder(true).
		SetBorderPadding(0, 0, 0, 1).
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(thunder.Colors.Accent).
		SetTitleColor(thunder.Colors.Accent).
		SetBackgroundColor(thunder.Colors.Background)

	return library
}

func (l *Library) readDir(path string) error {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	files := make([]File, 0, len(dirEntries))
	l.list.Grow(len(dirEntries))
	l.Frame.Clear()
	l.Clear()

	if len(dirEntries) != 0 {
		idx := 0
		for _, entry := range dirEntries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) != ".mp3" {
				continue
			}
			path := filepath.Join(path, entry.Name())
			name := strings.TrimSuffix(entry.Name(), ".mp3")

			file := NewFile(name, path, entry.IsDir())

			files = append(files, file)

			style, selectedStyle := l.GetStyle(file)
			l.list.InsertItem(idx, file, file.Name(), style, selectedStyle)
			idx++
		}
	}

	l.files = files
	l.currentPath = path
	dir := filepath.Base(l.currentPath)
	if dir != "/" {
		dir += "/"
	}
	l.Frame.AddText(" "+dir, true, tview.AlignLeft, thunder.Colors.Playlist)
	l.UpdateInfo()

	return nil
}

func (l *Library) Remove(idx int) error {
	file := l.files[idx]
	if err := os.Remove(file.Path()); err != nil {
		return err
	}

	l.list.RemoveItem(idx)
	l.files = append(l.files[:idx], l.files[idx+1:]...)

	if len(l.files) == 0 {
		return nil
	}

	if l.currentIdx > idx || l.currentIdx == len(l.files) {
		l.currentIdx--
	}

	return nil
}

// TODO: Make it more simple, log errors

func (l *Library) RemoveSelected() {
	selectedItems := l.list.GetSelectedItems()

	if len(selectedItems) == 0 {
		return
	}

	indices := make([]int, 0, len(selectedItems))

	for _, item := range selectedItems {
		for i, f := range l.files {
			if f.Path() == item.Path() {
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
		if err := l.Remove(idx); err != nil {
			continue
		}
	}
}

func (l *Library) RemovePlaylist(idx int) error {
	file := l.files[idx]
	if !file.IsDir() {
		return fmt.Errorf("not a directory")
	}

	if err := os.RemoveAll(file.Path()); err != nil {
		return err
	}

	l.list.RemoveItem(idx)
	l.files = append(l.files[:idx], l.files[idx+1:]...)

	if len(l.files) == 0 {
		return nil
	}

	if l.currentIdx > idx || l.currentIdx == len(l.files) {
		l.currentIdx--
	}

	return nil
}

func (l *Library) Clear() {
	l.currentIdx = 0
	l.list.Clear()
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
	return thunder.PlayPanel.playlist.GetModified()
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
