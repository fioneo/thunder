package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strconv"
)

type LibraryInfo struct {
	*tview.Box
	currentDir string
	selected   int
	copied     int
}

func NewLibraryInfo() *LibraryInfo {
	return &LibraryInfo{
		Box: tview.NewBox().
			SetBackgroundColor(thunder.Colors.Background),
	}
}

func (li *LibraryInfo) SetCurrentDir(dir string) { li.currentDir = dir }

func (li *LibraryInfo) SetSelectedCount(selected int) { li.selected = selected }

func (li *LibraryInfo) SetCopiedCount(copied int) { li.copied = copied }

func (li *LibraryInfo) Draw(screen tcell.Screen) {
	colors := thunder.Colors
	li.Box.DrawForSubclass(screen, li)

	x, y, width, height := li.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	base := tcell.StyleDefault.Background(colors.Background)

	printed := PrintText(screen, x, y, width, li.currentDir, base.Foreground(colors.Playlist))

	offsetRight := 0

	showSelected := li.selected > 0
	showCopied := li.copied > 0

	var selected, copied string

	if showSelected {
		selected = fmt.Sprintf(" %s ", strconv.Itoa(li.selected))
		offsetRight += len(selected)
	}

	if showCopied {
		copied = fmt.Sprintf(" %s ", strconv.Itoa(li.copied))
		offsetRight += len(copied)
	}

	bx := width - offsetRight + 1

	if showSelected && showCopied {
		bx -= 1 // gap
	}

	if x+printed >= bx {
		return
	} else {
		x = bx
	}

	if showSelected {
		printed := PrintText(screen, x, y, width, selected, base.Foreground(tcell.ColorBlack).Background(colors.Selected))

		x += printed + 1
	}

	if showCopied {
		PrintText(screen, x, y, width, copied, base.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow))
	}
}
