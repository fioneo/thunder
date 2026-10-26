package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type PlayBar struct {
	*tview.Box
	snap     Snapshot
	progress func() (pos, length time.Duration)
}

func NewPlayBar(progress func() (pos, length time.Duration)) *PlayBar {
	bar := &PlayBar{
		Box:      tview.NewBox(),
		progress: progress,
	}

	bar.SetBorder(true).
		SetTitle("Now Playing").
		SetBorderColor(thunder.Colors.Foreground).
		SetTitleColor(thunder.Colors.Foreground).
		SetBackgroundColor(thunder.Colors.Background)

	return bar
}

func (b *PlayBar) Render(s Snapshot) {
	if s.Version >= b.snap.Version {
		b.snap = s
	}
}

func (b *PlayBar) Draw(screen tcell.Screen) {
	b.Box.DrawForSubclass(screen, b)

	x, y, width, height := b.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	// padding
	if height <= 5 {
		y++
		height -= 2
	} else {
		y += 2
		height -= 4
	}
	x += 2
	width -= 4

	if width <= 0 || height <= 0 {
		return
	}

	colors := thunder.Colors
	base := tcell.StyleDefault.Background(colors.Background)
	s := b.snap

	// Now playing
	if s.Now == nil {
		text := "Nothing playing"
		x := x + (width-len(text))/2
		PrintText(screen, x, y, width, text, base.Foreground(colors.Foreground))
	} else {
		text := s.Now.Name()
		x := x + (width-len([]rune(text)))/2
		PrintText(screen, x, y, width, text, base.Foreground(colors.Highlighted))
	}
	y++

	// Progress
	if height >= 2 {
		pos, length := b.progress()
		b.drawProgress(screen, x, y, width, pos, length, base)
	}
	y++

	// Mode and volume
	if height >= 3 {
		volume := fmt.Sprintf("%d%%", volumeToPercentage(s.Volume))
		if s.Muted {
			volume = "muted"
		}
		icon := "\uf04c"
		if s.Paused {
			icon = "\uf04b"
		}

		info := fmt.Sprintf("%s %s %s", s.Mode.ToString(), icon, volume)
		x := x + (width-len(info))/2

		PrintText(screen, x, y, width, info, base.Foreground(colors.Foreground))
	}
}

func (b *PlayBar) drawProgress(screen tcell.Screen, x, y, width int, pos, length time.Duration, base tcell.Style) {
	colors := thunder.Colors
	left, right := fmtDuration(pos), fmtDuration(length)

	barWidth := width - len(left) - len(right) - 2
	if barWidth < 4 {
		PrintText(screen, x, y, width, left+" / "+right, base.Foreground(colors.Foreground))
		return
	}

	filled := 0
	if length > 0 {
		filled = int(int64(barWidth) * int64(pos) / int64(length))
	}
	filled = min(max(filled, 0), barWidth)

	PrintText(screen, x, y, width, left, base.Foreground(colors.Foreground))
	bx := x + len(left) + 1
	PrintText(screen, bx, y, filled, strings.Repeat("━", filled)+"│", base.Foreground(colors.Highlighted))
	PrintText(screen, bx+filled, y, barWidth-filled, strings.Repeat("─", barWidth-filled), base.Foreground(colors.Song))
	PrintText(screen, bx+barWidth+1, y, len(right), right, base.Foreground(colors.Foreground))
}
