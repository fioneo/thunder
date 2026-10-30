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
	flex        *tview.Flex
	pages       *tview.Pages
	info        *LibraryInfo
	list        *List
	files       []File
	copied      []File
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

	info := NewLibraryInfo()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(info, 1, 0, false).AddItem(pages, 0, 1, true)

	wrapper := tview.NewFrame(flex).SetBorders(0, 0, 0, 0, 0, 0)

	library := &Library{
		Frame:       wrapper,
		flex:        flex,
		pages:       pages,
		info:        info,
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

	library.list.SetSelectedFunc(func(count int) {
		info.SetSelectedCount(count)
	})

	library.SetInputCapture(library.handleKey)

	colors := thunder.Colors
	library.list.ApplySongStyles(colors).SetBackgroundColor(colors.Background)

	library.
		SetBorderPadding(0, 0, 1, 1).
		SetBorder(true).
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

	l.info.SetCurrentDir(dir)
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

	if len(l.copied) == 0 {
		return err
	}

	for idx, f := range l.copied {
		if f.Path() == file.Path() {
			l.copied = append(l.copied[:idx], l.copied[idx+1:]...)
		}
	}

	return err
}

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

	// This is crucial, because removing in ascending order shifts elements,
	// and eventually we can get an index out of range
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

	c := make(map[string]struct{}, len(l.copied))

	for _, file := range l.copied {
		c[file.Path()] = struct{}{}
	}

	files := l.selectedFiles()

	copied := 0

	if len(files) != 0 {
		for _, file := range files {
			if _, ok := c[file.Path()]; !ok {
				l.copied = append(l.copied, file)
				copied++
			}
		}
	} else {
		file := l.files[idx]
		if _, ok := c[file.Path()]; !ok {
			l.copied = append(l.copied, file)
			copied++
		}
	}

	return copied
}

func (l *Library) Paste() int {
	files := l.copied
	path := l.currentPath

	pasted := 0

	for _, file := range files {
		switch file.IsDir() {
		case true:
			if ok := l.PasteDir(file, path); ok {
				pasted++
			}
		case false:
			if ok := l.PasteFile(file, path); ok {
				pasted++
			}
		}
	}

	return pasted
}

func (l *Library) PasteFile(file File, path string) bool {
	from, err := os.Open(file.Path())
	if err != nil {
		SendErr(err)
		return false
	}
	defer from.Close()

	toPath := filepath.Join(path, filepath.Base(from.Name()))
	to, err := os.OpenFile(toPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		SendErr(err)
		return false
	}
	defer to.Close()

	if _, err := io.Copy(to, from); err != nil {
		errorPopup(fmt.Errorf("Copy error\nSee log file for details."))
		SendErr(fmt.Errorf("copy from %s to %s: %v", file.Path(), toPath, err))
		return false
	}

	return true
}

func (l *Library) PasteDir(fromDir File, path string) bool {
	dirEntries, err := os.ReadDir(fromDir.Path())
	if err != nil {
		SendErr(err)
		return false
	}

	toDir := filepath.Join(
		path,
		filepath.Base(fromDir.Path()),
	)

	_ = os.RemoveAll(toDir)

	err = os.Mkdir(
		toDir,
		0755,
	)

	if err != nil {
		SendErr(err)
		return false
	}

	for _, entry := range dirEntries {
		path := filepath.Join(fromDir.Path(), entry.Name())
		name := entry.Name()

		file := NewFile(name, path, entry.IsDir())

		if entry.IsDir() {
			l.PasteDir(file, toDir)
		} else {
			l.PasteFile(file, toDir)
		}
	}

	return true
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
	l.UpdateInfo()
}

func (l *Library) selectedFiles() []File {
	selected := l.list.GetSelectedItems()
	files := make([]File, len(selected))
	for i, item := range selected {
		files[i] = item.(File)
	}
	return files
}

func (l *Library) Clear() {
	l.currentIdx = 0
	l.list.Clear()
	l.files = nil
}

func (l *Library) Refresh() {
	if l.currentIdx < 0 || l.currentIdx >= len(l.files) {
		l.readDir(l.currentPath)
	} else {
		file := l.files[l.currentIdx]
		l.readDir(l.currentPath)
		l.SetCurrentIdx(l.GetFileIdx(file.Path()))
	}

	l.UpdateInfo()
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
		if len(l.copied) != 0 {
			l.copied = nil
			l.info.SetCopiedCount(0)
		}
	case tcell.KeyCtrlC:
		l.Copy()
		l.Refresh()
		l.info.SetCopiedCount(len(l.copied))
	case tcell.KeyCtrlV:
		l.Paste()
		l.copied = nil
		l.Refresh()
		l.info.SetCopiedCount(0)
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
			return e
		}
		l.idxHistory[currentPath] = idx
	case 'h':
		path := filepath.Dir(l.currentPath)

		if err := l.readDir(path); err != nil {
			thunder.ErrCh <- err
			return e
		}

		if idx, ok := l.idxHistory[path]; ok {
			l.SetCurrentIdx(idx)
		}
	case 'd':
		selectedItems := l.list.GetSelectedItems()
		if len(selectedItems) > 0 {
			popupText := fmt.Sprintf("Are you sure you want to delete the selected files?\nNumber of files to be deleted: %d", len(selectedItems))
			confirmationPopup(popupText, func(_ int, label string) {
				if label == "yes" {
					l.RemoveSelected()
					l.list.ClearSelected()
					l.info.SetCopiedCount(len(l.copied))
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

		text := fmt.Sprintf("[ %s ]\nAre you sure you want to delete the current %s?", file.Name(), name)

		confirmationPopup(text, func(_ int, label string) {
			if label == "yes" {
				if err := l.Remove(idx); err != nil {
					SendErr(err)
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
				return
			}

			l.Refresh()
			idx := l.GetFileIdx(newPath)
			l.SetCurrentIdx(idx)
		})
	case 'n':
		name := "New Directory"

		inputPopup("Create directory", name, name, func(ok bool, dirName string) {
			if !ok {
				return
			}

			name = dirName
			path := filepath.Join(l.currentPath, name)

			if err := os.Mkdir(path, 0755); err != nil {
				SendErr(err)
				return
			}

			l.Refresh()
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
