package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Selectable interface {
	Path() string
}

type Selection struct {
	items []Selectable
	set   map[string]struct{}
}

func NewSelection() *Selection {
	return &Selection{
		set: make(map[string]struct{}),
	}
}

func (s *Selection) Add(item Selectable) {
	if !s.Exist(item) {
		s.items = append(s.items, item)
		s.set[item.Path()] = struct{}{}
	}
}

func (s *Selection) Remove(item Selectable) {
	delete(s.set, item.Path())
	for idx, i := range s.items {
		if i.Path() == item.Path() {
			s.items = append(s.items[:idx], s.items[idx+1:]...)
			return
		}
	}
}

func (s *Selection) Length() int {
	return len(s.items)
}

func (s *Selection) Exist(item Selectable) bool {
	_, ok := s.set[item.Path()]
	return ok
}

func (s *Selection) Clear() {
	s.items = nil
	s.set = make(map[string]struct{})
}

func (s *Selection) Toggle(item Selectable) {
	if s.Exist(item) {
		s.Remove(item)
		return
	}

	s.Add(item)
}

func (s *Selection) Retain(keep func(Selectable) bool) {
	kept := s.items[:0]
	for _, item := range s.items {
		if keep(item) {
			kept = append(kept, item)
		} else {
			delete(s.set, item.Path())
		}
	}
	clear(s.items[len(kept):])
	s.items = kept
}

type listItem struct {
	item                   Selectable
	text                   string
	style                  *tcell.Style
	selectedStyle          *tcell.Style
	highlightStyle         *tcell.Style
	highlightSelectedStyle *tcell.Style
}

type List struct {
	*tview.Box

	items []*listItem

	selection *Selection

	anchorIdx int

	currentIdx int

	textStyle tcell.Style

	currentItemStyle tcell.Style

	selectedStyle tcell.Style

	selectedFocusOnly bool

	highlightFullLine bool

	wrapAround bool

	itemOffset int

	horizontalOffset int

	changed func(index int, text string)

	selected func(selectedCount int)
}

func NewList() *List {
	return &List{
		Box:        tview.NewBox().SetBorderPadding(0, 0, 0, 1),
		selection:  NewSelection(),
		currentIdx: 0,
		anchorIdx:  -1,
	}
}

func (l *List) Grow(capacity int) *List {
	l.items = make([]*listItem, 0, capacity)
	return l
}

func (l *List) SetItemHighlightedStyle(idx int, highlightStyle *tcell.Style, highlightSelectedStyle *tcell.Style) {
	if idx < 0 || idx >= len(l.items) {
		return
	}

	listItem := l.items[idx]

	listItem.highlightStyle = highlightStyle
	listItem.highlightSelectedStyle = highlightSelectedStyle
}

func (l *List) SetItemOffset(items int) *List {
	l.itemOffset = items
	return l
}

func (l *List) GetItemOffset() int {
	return l.itemOffset
}

func (l *List) SetCurrentIdx(idx int) *List {
	if idx < 0 || idx >= len(l.items) {
		return l
	}
	l.currentIdx = idx
	l.adjustOffset()
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

func (l *List) SetSelectedFunc(handler func(selectedCount int)) {
	l.selected = handler
}

func (l *List) GetSelectedItems() []Selectable {
	return l.selection.items
}

func (l *List) InsertItem(idx int, item Selectable, text string, style, selectedStyle *tcell.Style) *List {
	listItem := &listItem{
		item:          item,
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

	// if l.currentIdx < len(l.items) && l.currentIdx >= idx {
	// 	l.currentIdx++
	// }

	l.items = append(l.items, nil)
	if idx < len(l.items)-1 {
		copy(l.items[idx+1:], l.items[idx:])
	}
	l.items[idx] = listItem

	if len(l.items) == 1 && l.changed != nil {
		item := l.items[0]
		l.changed(0, item.text)
	}

	return l

}

func (l *List) RemoveItem(idx int) *List {
	if len(l.items) == 0 {
		return l
	}

	if idx < 0 {
		idx = len(l.items) + idx
	}
	if idx >= len(l.items) {
		idx = len(l.items) - 1
	}
	if idx < 0 {
		idx = 0
	}

	item := l.items[idx].item

	l.selection.Remove(item)
	l.items = append(l.items[:idx], l.items[idx+1:]...)

	if len(l.items) == 0 {
		return l
	}

	previousCurrentIdx := l.currentIdx
	if l.currentIdx > idx || l.currentIdx == len(l.items) {
		l.currentIdx--
	}

	if previousCurrentIdx == idx && l.changed != nil {
		item := l.items[l.currentIdx]
		l.changed(l.currentIdx, item.text)
	}

	return l
}

type ListEntry struct {
	Item Selectable
	Text string
}

func (l *List) Reset(entries []ListEntry) {
	present := make(map[string]struct{}, len(entries))
	items := make([]*listItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, &listItem{item: e.Item, text: e.Text})
		present[e.Item.Path()] = struct{}{}
	}
	l.items = items

	l.selection.Retain(func(item Selectable) bool {
		_, ok := present[item.Path()]
		return ok
	})
	l.anchorIdx = -1

	l.currentIdx = min(max(l.currentIdx, 0), max(len(items)-1, 0))
	l.adjustOffset()
}

func (l *List) ApplySongStyles(c *Colors) *List {
	return l.
		SetTextStyle(tcell.StyleDefault.Foreground(c.Song).Background(c.Background)).
		SetSelectedFocusOnly(true).
		SetCurrentItemStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(c.Song)).
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(c.Selected)).
		SetHighlightFullLine(true).
		SetWrapAround(false)
}

func (l *List) GetItemCount() int {
	return len(l.items)
}

func (l *List) Clear() *List {
	l.ClearSelected()
	l.items = nil
	l.currentIdx = 0
	return l
}

func (l *List) ClearSelected() {
	l.selection.Clear()
	l.anchorIdx = -1
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

func (l *List) SetCurrentItemStyle(style tcell.Style) *List {
	l.currentItemStyle = style
	return l

}

func (l *List) SetSelectedStyle(style tcell.Style) *List {
	l.selectedStyle = style
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
	for index, listItem := range l.items {
		if index < l.itemOffset {
			continue
		}

		if y == bottomLimit {
			break
		}

		checkFocus := !l.selectedFocusOnly || l.HasFocus()

		current := index == l.currentIdx && checkFocus
		style := l.textStyle
		if listItem.style != nil {
			style = *listItem.style
		}

		selectedStyle := l.selectedStyle
		selected := l.selection.Exist(listItem.item)

		if current {
			if listItem.selectedStyle != nil {
				style = *listItem.selectedStyle
			} else {
				style = l.currentItemStyle
			}
		}

		highlightStyle := tcell.StyleDefault
		highlightSelectedStyle := tcell.StyleDefault
		highlighted := listItem.highlightStyle != nil && listItem.highlightSelectedStyle != nil

		if highlighted {
			highlightStyle = *listItem.highlightStyle
			highlightSelectedStyle = *listItem.highlightSelectedStyle
			if current {
				style = highlightSelectedStyle
			} else {
				style = highlightStyle
			}
		}

		text := listItem.text
		printedWidth := printListItem(screen, text, x, y, l.horizontalOffset, width, style, selectedStyle, selected, highlightStyle, highlighted)
		if printedWidth > maxWidth {
			maxWidth = printedWidth
		}

		if (current || selected) && l.highlightFullLine {
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
		if l.currentIdx < 0 ||
			l.currentIdx >= len(l.items) ||
			len(l.items) == 0 {
			return
		}
		previousIdx := l.currentIdx
		listItem := l.items[l.currentIdx]

		switch e.Key() {
		case tcell.KeyCtrlA:
			l.updateSelectionRange(0, len(l.items)-1, true)
		case tcell.KeyDown:
			if e.Modifiers()&tcell.ModShift != 0 {
				if l.anchorIdx == -1 {
					l.anchorIdx = previousIdx
				}
				l.currentIdx++
				l.updateSelectionRange(l.anchorIdx, l.currentIdx, true)
			} else {
				l.anchorIdx = -1
				l.currentIdx++
			}
		case tcell.KeyUp:
			if e.Modifiers()&tcell.ModShift != 0 {
				if l.anchorIdx == -1 {
					l.anchorIdx = previousIdx
				}
				l.currentIdx--
				l.updateSelectionRange(l.anchorIdx, l.currentIdx, true)
			} else {
				l.anchorIdx = -1
				l.currentIdx--
			}
		case tcell.KeyCtrlJ:
			if l.anchorIdx == -1 {
				l.anchorIdx = previousIdx
			}
			l.currentIdx = len(l.items) - 1
			l.updateSelectionRange(l.anchorIdx, l.currentIdx, true)
		case tcell.KeyCtrlK:
			if l.anchorIdx == -1 {
				l.anchorIdx = previousIdx
			}
			l.currentIdx = 0
			l.updateSelectionRange(l.anchorIdx, l.currentIdx, true)
		}
		switch e.Rune() {
		case 'j':
			l.anchorIdx = -1
			l.currentIdx++
		case 'k':
			l.anchorIdx = -1
			l.currentIdx--
		case 'g':
			l.anchorIdx = -1
			l.currentIdx = 0
		case 'G':
			l.anchorIdx = -1
			l.currentIdx = len(l.items) - 1
		case 's':
			l.selection.Toggle(listItem.item)
		case 'J':
			if l.anchorIdx == -1 {
				l.anchorIdx = previousIdx
			}
			l.currentIdx++
			l.updateSelectionRange(l.anchorIdx, l.currentIdx, true)
		case 'K':
			if l.anchorIdx == -1 {
				l.anchorIdx = previousIdx
			}
			l.currentIdx--
			l.updateSelectionRange(l.anchorIdx, l.currentIdx, true)
		}

		if l.currentIdx < 0 {
			if l.wrapAround {
				l.currentIdx = len(l.items) - 1
			} else {
				l.currentIdx = 0
			}
		} else if l.currentIdx >= len(l.items) {
			if l.wrapAround {
				l.currentIdx = 0
			} else {
				l.currentIdx = len(l.items) - 1
			}
		}

		if l.currentIdx != previousIdx && l.currentIdx < len(l.items) {
			if l.changed != nil {
				l.changed(l.currentIdx, l.items[l.currentIdx].text)
			}
			l.adjustOffset()
		}

		if l.selected != nil {
			l.selected(len(l.selection.items))
		}
	})
}

func (l *List) updateSelectionRange(anchor, current int, add bool) {
	start := min(anchor, current)
	end := max(anchor, current)
	for idx := start; idx <= end; idx++ {
		if idx < 0 || idx >= len(l.items) {
			continue
		}
		item := l.items[idx].item
		if add {
			l.selection.Add(item)
		} else {
			l.selection.Remove(item)
		}
	}
}

// func (l *List) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
// 	return l.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
// 		if !l.InRect(event.Position()) {
// 			return false, nil
// 		}
//
// 		switch action {
// 		case tview.MouseScrollUp:
// 			l.itemOffset--
// 			consumed = true
// 		case tview.MouseScrollDown:
// 			l.itemOffset++
// 			consumed = true
// 		case tview.MouseScrollLeft:
// 			l.horizontalOffset--
// 			consumed = true
// 		case tview.MouseScrollRight:
// 			l.horizontalOffset++
// 			consumed = true
// 		}
//
// 		return
// 	})
// }
//
