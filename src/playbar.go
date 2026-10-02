package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Playbar struct {
	*tview.Frame
}

func NewPlaybar() *Playbar {
	playbar := &Playbar{
		Frame: tview.NewFrame(tview.NewBox()),
	}

	playbar.SetBorder(true).
		SetBorderColor(tcell.ColorBlue).
		SetTitle("Now Playing").
		SetTitleColor(tcell.ColorBlue).
		SetBorderPadding(0, 0, 1, 1)

	return playbar
}
