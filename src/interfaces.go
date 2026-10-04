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

type Node interface {
	IsDir() bool
	Children() []Node
	Name() string
	Path() string
	Parent() Node
	SetIsDir(bool)
	SetChildren([]Node)
	SetName(string)
	SetPath(string)
	SetParent(Node)
}
