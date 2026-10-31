package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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

type Popup interface {
	ID() string
	Show()
	Hide()
	Primitive() tview.Primitive
	Focusable() bool
	Focus()
	Blur()
}
