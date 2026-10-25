package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io"
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
	copied      []string
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
		library.UpdateInfo()
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

	var err error
	if !file.IsDir() {
		err = os.Remove(file.Path())
	} else {
		err = os.RemoveAll(file.Path())
	}

	l.list.RemoveItem(idx)
	l.files = append(l.files[:idx], l.files[idx+1:]...)

	if len(l.files) == 0 {
		return err
	}

	if l.currentIdx > idx || l.currentIdx == len(l.files) {
		l.currentIdx--
	}

	return err
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

func (l *Library) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (l *Library) Copy() int {
	idx := l.currentIdx
	if len(l.files) == 0 ||
		idx < 0 ||
		idx >= len(l.files) {
		return 0
	}

	paths := l.selectedPaths()

	copied := 0

	if len(paths) != 0 {
		l.copied = append(l.copied, paths...)
		copied += len(paths)
	} else {
		l.copied = append(l.copied, l.files[idx].Path())
		copied += 1
	}

	return copied
}

func (l *Library) Paste() int {
	if len(l.copied) == 0 {
		return 0
	}

	paths := l.copied

	pasted := 0
	for _, path := range paths {
		from, err := os.Open(path)
		if err != nil {
			SendErr(err)
			continue
		}

		toPath := filepath.Join(l.currentPath, filepath.Base(from.Name()))
		to, err := os.OpenFile(toPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			SendErr(err)
			continue
		}

		if _, err := io.Copy(to, from); err != nil {
			err := fmt.Errorf("copy from %s to %s: %v", path, toPath, err)
			errorPopup(err)
			SendErr(err)
			continue
		}
		from.Close()
		to.Close()
		pasted++
	}

	return pasted
}

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

func (l *Library) selectedPaths() []string {
	selected := l.list.GetSelectedItems()
	paths := make([]string, len(selected))
	for i, item := range selected {
		paths[i] = item.Path()
	}
	return paths
}

func (l *Library) Clear() {
	l.currentIdx = 0
	l.list.Clear()
	l.files = nil
}

func (l *Library) Refresh() {
	if l.currentIdx < 0 || l.currentIdx >= len(l.files) {
		return
	}
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
	case tcell.KeyCtrlC:
		copied := l.Copy()

		word := "file"

		if copied > 1 {
			word = "files"
		}

		if copied > 0 {
			infoPopup(fmt.Sprintf(
				"Copied %d %s",
				copied,
				word,
			))
		}
		l.UpdateInfo()
	case tcell.KeyCtrlV:
		pasted := l.Paste()

		word := "file"

		if pasted > 1 {
			word = "files"
		}

		if pasted > 0 {
			infoPopup(fmt.Sprintf(
				"Pasted %d %s",
				pasted,
				word,
			))
		}
		l.UpdateInfo()
	}
	switch e.Rune() {
	case 'l':
		selectedItems := l.list.GetSelectedItems()

		if len(selectedItems) > 0 {
			files := make([]File, 0, len(selectedItems))

			for _, item := range selectedItems {
				files = append(files, item.(File))
			}

			songs := filesToSongs(files)
			thunder.Playback.Enqueue(songs...)
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
			thunder.Playback.Enqueue(song)
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

		name := "file"

		if file.IsDir() {
			name = "directory"
		}

		text := fmt.Sprintf("[ %s ]\nAre you sure you want to delete the current %s?", name, file.Name())

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
		style = l.dirStyle()
		selectedStyle = l.dirSelectedStyle()
	}

	return &style, &selectedStyle
}

func (l *Library) songStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)
}

func (l *Library) songSelectedStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)
}

func (l *Library) dirStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(thunder.Colors.Playlist).Background(thunder.Colors.Background)
}

func (l *Library) dirSelectedStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Playlist)
}
