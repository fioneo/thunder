package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Song interface {
	Name() string
	Path() string
}

type PlaylistInfo interface {
	Name() string
	Path() string
	Songs() []Song
}

type Panel interface {
	HasFocus() bool
	SetBorderColor(tcell.Color) *tview.Box
	SetTitleColor(tcell.Color) *tview.Box
}

type File interface {
	IsDir() bool
	Name() string
	Path() string
}
