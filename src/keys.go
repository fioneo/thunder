package main

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

const seekStep = 5 * time.Second

func (t *Thunder) handlePlaybackKey(e *tcell.EventKey) *tcell.EventKey {
	if e.Key() != tcell.KeyRune {
		return e
	}

	c := t.Playback
	switch e.Rune() {
	case 'p':
		c.TogglePause()
	case '>':
		c.Next()
	case '<':
		c.Previous()
	case 'x':
		c.Stop()
	case ']':
		c.Seek(seekStep)
	case '[':
		c.Seek(-seekStep)
	case '+', '=':
		c.ChangeVolume(volumeStep)
	case '-':
		c.ChangeVolume(-volumeStep)
	case 'm':
		c.ToggleMute()
	case 'c':
		c.CycleMode()
	default:
		return e
	}
	return nil
}
