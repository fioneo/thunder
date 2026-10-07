package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type listItem struct {
	text          string
	style         *tcell.Style
	selectedStyle *tcell.Style
}

type List struct {
	*tview.Box

	items []*listItem

	currentIdx int

	textStyle tcell.Style

	selectedTextStyle tcell.Style

	multiplySelectionStyle tcell.Style

	selectedFocusOnly bool

	highlightFullLine bool

	wrapAround bool

	itemOffset int

	horizontalOffset int

	changed func(index int, text string)

	selected func(index int, text string)
}

func NewList() *List {
	return &List{
		Box: tview.NewBox(),
	}
}

func (l *List) SetItemOffset(items int) *List {
	l.itemOffset = items
	return l
}

func (l *List) SetCurrentIdx(idx int) *List {
	if idx < 0 && idx >= len(l.items) {
		return l
	}
	l.currentIdx = idx
	return l
}

func (l *List) GetCurrentIdx() int {
	return l.currentIdx
}

func (l *List) SetSelectedFocusOnly(focusOnly bool) *List {
	l.selectedFocusOnly = focusOnly
	return l
}

func (l *List) SetHighlightFullLine(highlight bool) *List {
	l.highlightFullLine = highlight
	return l
}

func (l *List) SetWrapAround(wrapAround bool) *List {
	l.wrapAround = wrapAround
	return l
}

func (l *List) SetChangedFunc(handler func(idx int, text string)) *List {
	l.changed = handler
	return l
}

func (l *List) SetSelectedFunc(handler func(idx int, text string)) *List {
	l.selected = handler
	return l
}

func (l *List) InsertItem(idx int, text string, style, selectedStyle *tcell.Style) *List {
	item := &listItem{
		text:          text,
		style:         style,
		selectedStyle: selectedStyle,
	}

	if idx < 0 {
		idx = len(l.items) + idx + 1
	}
	if idx < 0 {
		idx = 0
	} else if idx > len(l.items) {
		idx = len(l.items)
	}

	if l.currentIdx < len(l.items) && l.currentIdx >= idx {
		l.currentIdx++
	}

	l.items = append(l.items, nil)
	if idx < len(l.items)-1 {
		copy(l.items[idx+1:], l.items[idx:])
	}
	l.items[idx] = item

	// if len(l.items) == 1 && l.changed != nil {
	// 	item := l.items[0]
	// 	l.changed(0, item.MainText, item.SecondaryText, item.Shortcut)
	// }

	return l

}

func (l *List) RemoveItem(idx int) *List {
	if len(l.items) == 0 {
		return l
	}

	// Adjust index.
	if idx < 0 {
		idx = len(l.items) + idx
	}
	if idx >= len(l.items) {
		idx = len(l.items) - 1
	}
	if idx < 0 {
		idx = 0
	}

	l.items = append(l.items[:idx], l.items[idx+1:]...)

	if len(l.items) == 0 {
		return l
	}

	// previousCurrentItem := l.currentIdx
	if l.currentIdx > idx || l.currentIdx == len(l.items) {
		l.currentIdx--
	}

	// if previousCurrentItem == index && l.changed != nil {
	// 	item := l.items[l.currentItem]
	// 	l.changed(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
	// }

	return l
}

func (l *List) GetItemCount() int {
	return len(l.items)
}

func (l *List) Clear() *List {
	l.items = nil
	l.currentIdx = 0
	return l
}

func (l *List) adjustOffset() {
	_, _, _, height := l.GetInnerRect()
	if height == 0 {
		return
	}
	if l.currentIdx < l.itemOffset {
		l.itemOffset = l.currentIdx
	} else {
		if l.currentIdx-l.itemOffset >= height {
			l.itemOffset = l.currentIdx + 1 - height
		}
	}
}

// styles

func (l *List) SetTextStyle(style tcell.Style) *List {
	l.textStyle = style
	return l
}

func (l *List) SetSelectedStyle(style tcell.Style) *List {
	l.selectedTextStyle = style
	return l

}

func (l *List) SetMultiplySelectionStyle(style tcell.Style) *List {
	l.multiplySelectionStyle = style
	return l
}

func (l *List) Draw(screen tcell.Screen) {
	l.Box.DrawForSubclass(screen, l)

	x, y, width, height := l.GetInnerRect()
	if height == 0 {
		return
	}
	bottomLimit := y + height
	_, totalHeight := screen.Size()
	if bottomLimit > totalHeight {
		bottomLimit = totalHeight
	}

	if l.itemOffset >= len(l.items) {
		l.itemOffset = len(l.items) - 1
	}

	if l.horizontalOffset < 0 {
		l.horizontalOffset = 0
	}

	if len(l.items)-l.itemOffset < height {
		l.itemOffset = len(l.items) - height
	}

	if l.itemOffset < 0 {
		l.itemOffset = 0
	}

	var maxWidth int
	for index, item := range l.items {
		if index < l.itemOffset {
			continue
		}

		if y == bottomLimit {
			break
		}

		selected := index == l.currentIdx && (!l.selectedFocusOnly || l.HasFocus())
		style := l.textStyle
		if item.style != nil {
			style = *item.style
		}
		if selected {
			if item.selectedStyle != nil {
				style = *item.selectedStyle
			} else {
				style = l.selectedTextStyle
			}
		}

		text := item.text
		_, _, printedWidth := printWithStyle(screen, text, x, y, l.horizontalOffset, width, style, false)
		if printedWidth > maxWidth {
			maxWidth = printedWidth
		}

		if selected && l.highlightFullLine {
			for bx := printedWidth; bx < width; bx++ {
				screen.SetContent(x+bx, y, ' ', nil, style)
			}
		}

		y++
		if y >= bottomLimit {
			break
		}
	}

	if l.horizontalOffset > 0 && maxWidth < width {
		l.horizontalOffset -= width - maxWidth
		l.Draw(screen)
	}
}

func (l *List) InputHandler() func(e *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return l.WrapInputHandler(func(e *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if len(l.items) == 0 {
			return
		}
		previousIdx := l.currentIdx

		switch e.Rune() {
		case 'j':
			l.currentIdx++
		case 'k':
			l.currentIdx--
		}

		if l.currentIdx < 0 {
			l.currentIdx = 0
		} else if l.currentIdx >= len(l.items) {
			l.currentIdx = len(l.items) - 1
		}

		if l.currentIdx != previousIdx && l.currentIdx < len(l.items) {
			if l.changed != nil {
				item := l.items[l.currentIdx]
				l.changed(l.currentIdx, item.text)
			}
			l.adjustOffset()
		}
	})
}
