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
	pages       *tview.Pages
	list        *List
	files       []File
	currentPath string
	currentIdx  int
	idxHistory  map[string]int
}

func NewLibrary(path string) *Library {
	list := NewList()
	emptyDirectory := tview.NewTextView().
		SetText(" empty directory").
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background))

	pages := tview.NewPages().AddPage("library", list, true, true).AddPage("empty_directory", emptyDirectory, true, false)
	frame := tview.NewFrame(pages).SetBorders(0, 0, 0, 0, 0, 0)

	library := &Library{
		Frame:       frame,
		pages:       pages,
		list:        list,
		currentPath: path,
		idxHistory:  make(map[string]int),
	}

	if err := library.readDir(path); err != nil {
		exit(err)
	}

	library.list.SetChangedFunc(func(idx int, _ string) {
		library.currentIdx = idx
	})

	library.SetInputCapture(library.handleKey)

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
			SendErr(err)
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

func (l *Library) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// func (l *Library) RenameDir(oldPath, name string) error {
// 	newPath := filepath.Join(filepath.Dir(oldPath), name)
// 	return os.Rename(oldPath, newPath)
// }

func (l *Library) GetFileIdx(path string) int {
	for i, f := range l.files {
		if f.Path() == path {
			return i
		}
	}
	return -1
}

func (l *Library) SetCurrentIdx(idx int) {
	if idx < 0 || idx >= len(l.files) {
		return
	}
	l.currentIdx = idx
	l.list.SetCurrentIdx(idx)
}

func (l *Library) Clear() {
	l.currentIdx = 0
	l.list.Clear()
	l.files = nil
}

func (l *Library) Refresh() {
	file := l.files[l.currentIdx]
	l.readDir(l.currentPath)
	l.SetCurrentIdx(l.GetFileIdx(file.Path()))

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

	l.CheckIsEmpty()
}

func (l *Library) CheckIsModified() bool { return thunder.PlayPanel.playlist.GetModified() }

func (l *Library) CheckIsEmpty() {
	if len(l.files) == 0 {
		l.pages.SwitchToPage("empty_directory")
	} else {
		l.pages.SwitchToPage("library")
	}
}

func (l *Library) handleKey(e *tcell.EventKey) *tcell.EventKey {
	idx := l.currentIdx

	switch e.Key() {
	case tcell.KeyCtrlR:
		l.Refresh()
	case tcell.KeyEscape:
		if len(l.list.GetSelectedItems()) != 0 {
			l.list.ClearSelected()
			l.UpdateInfo()
		}
	}
	switch e.Rune() {
	case 'l':
		selectedItems := l.list.GetSelectedItems()

		if len(selectedItems) > 0 {
			files := make([]File, 0, len(selectedItems))

			for _, item := range selectedItems {
				files = append(files, item.(File))
			}

			songs, ok := filesToSongs(files)
			if !ok {
				return e // Ignore directories
			}
			thunder.PlayPanel.playlist.InsertAll(songs)
			l.list.ClearSelected()
			l.UpdateInfo()
			return e
		}

		if len(l.files) == 0 ||
			idx < 0 ||
			idx >= len(l.files) {
			return e
		}

		file := l.files[idx]

		if !file.IsDir() {
			song := fileToSong(file)
			thunder.PlayPanel.playlist.Insert(song)
			return e
		}

		currentPath := l.currentPath
		path := file.Path()

		if err := l.readDir(path); err != nil {
			SendErr(err)
			errorPopup(err)
			return e
		}
		l.idxHistory[currentPath] = idx
	case 'h':
		path := filepath.Dir(l.currentPath)

		if err := l.readDir(path); err != nil {
			thunder.ErrCh <- err
			errorPopup(err)
			return e
		}

		if idx, ok := l.idxHistory[path]; ok {
			if idx < 0 || idx >= len(l.files) {
				return e
			}
			l.currentIdx = idx
			l.list.SetCurrentIdx(idx)
		}
	case 'd':
		selectedItems := l.list.GetSelectedItems()
		if len(selectedItems) > 0 {
			popupText := fmt.Sprintf("Are you sure you want to delete the selected files?\nNumber of files to be deleted: %d", len(selectedItems))
			confirmationPopup(popupText, func(_ int, label string) {
				if label == "yes" {
					l.RemoveSelected()
					l.list.ClearSelected()
					l.UpdateInfo()
				}
			})
			return e
		}

		if len(l.files) == 0 ||
			idx < 0 ||
			idx >= len(l.files) {
			return e
		}

		file := l.files[idx]

		if file.IsDir() {
			text := fmt.Sprintf("[ %s ]\nAre you sure you want to delete the current folder?\nThis will delete all files in it.", file.Name())
			confirmationPopup(text, func(_ int, label string) {
				if label == "yes" {
					if err := l.RemovePlaylist(idx); err != nil {
						SendErr(err)
						errorPopup(err)
					}
				}
			})
			return e
		}

		text := fmt.Sprintf("[ %s ]\nAre you sure you want to delete the current file?", file.Name())

		confirmationPopup(text, func(_ int, label string) {
			if label == "yes" {
				if err := l.Remove(idx); err != nil {
					SendErr(err)
					errorPopup(err)
				}
			}
		})

		l.UpdateInfo()
	case 'r':
		if len(l.files) == 0 ||
			idx < 0 ||
			idx >= len(l.files) {
			return e
		}
		file := l.files[idx]

		inputPopup("Rename file", file.Name(), file.Name(), func(ok bool, name string) {
			if !ok {
				return
			}
			if !file.IsDir() && !strings.HasSuffix(name, ".mp3") {
				name += ".mp3"
			}
			newPath := filepath.Join(filepath.Dir(file.Path()), name)

			if err := l.Rename(file.Path(), newPath); err != nil {
				SendErr(err)
				errorPopup(err)
				return
			}

			l.Refresh()
			idx := l.GetFileIdx(newPath)
			l.SetCurrentIdx(idx)
		})
	case 'o':
		if l.CheckIsModified() {
			confirmationPopup("You have unsaved changes.\nAre you sure you want to close the current playlist?", func(_ int, label string) {
				if label == "yes" {
					playlistInfo, ok := directoryToPlaylistInfo(filepath.Base(l.currentPath), l.currentPath, l.files)
					if !ok {
						err := fmt.Errorf("directory containts other directories")
						errorPopup(err)
						SendErr(err)
						return
					}

					thunder.PlayPanel.UpdateInfo(playlistInfo)
				}
			})
		} else {
			playlistInfo, ok := directoryToPlaylistInfo(filepath.Base(l.currentPath), l.currentPath, l.files)

			if !ok {
				err := fmt.Errorf("directory containts other directories")
				errorPopup(err)
				SendErr(err)
				return e
			}

			thunder.PlayPanel.UpdateInfo(playlistInfo)
		}
	case 'n':
		if l.CheckIsModified() {
			confirmationPopup("You have unsaved changes.\nAre you sure you want to create a new playlist?", func(_ int, label string) {
				if label == "yes" {
					playlistInfo := NewDefaultPlaylistInfo(l.currentPath)

					thunder.PlayPanel.UpdateInfo(playlistInfo)
				}
			})
		} else {
			playlistInfo := NewDefaultPlaylistInfo(l.currentPath)

			thunder.PlayPanel.UpdateInfo(playlistInfo)
		}
	}
	return e
}

func (l *Library) GetCurrentPath() string {
	return l.currentPath
}

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
