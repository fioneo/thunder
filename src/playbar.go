package main

import (
	"github.com/rivo/tview"
)

type PlayBar struct {
	*tview.Frame
}

func NewPlayBar() *PlayBar {
	playbar := &PlayBar{
		Frame: tview.NewFrame(tview.NewBox().SetBackgroundColor(thunder.Colors.Background)),
	}

	playbar.SetBorder(true).
			SetTitle("Now Playing").
			SetBorderColor(thunder.Colors.Foreground).
			SetTitleColor(thunder.Colors.Foreground).
			SetBorderPadding(0, 0, 1, 1).
			SetBackgroundColor(thunder.Colors.Background)

	return playbar
}
