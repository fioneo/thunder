package main

import (
	"fmt"
	"sync"
	"time"
)

// PlaybackEngine is implemented by Player.
type PlaybackEngine interface {
	// Play starts playing the song and calls onFinished when it ends.
	Play(song Song, onFinished func()) error
	// Stop stops the current song.
	Stop()
	// TogglePause pauses the current song or resumes it if it is paused.
	TogglePause()
	// Seek moves the position of the current song by delta (negative to go back).
	Seek(delta time.Duration) error
	// Position returns the current position and the length of the song.
	Position() (pos, length time.Duration)
	// ChangeVolume changes the volume by delta.
	ChangeVolume(delta float64)
	// ToggleMute mutes the sound or unmutes it if it is muted.
	ToggleMute()
	// Volume returns the current volume level and whether it is muted.
	Volume() (level float64, muted bool)
}

// Snapshot is an immutable copy of the controller state, used by the UI.
type Snapshot struct {
	// Version increases on every change of the controller state.
	Version uint64
	// Now is the current song, or nil if nothing is playing.
	Now Song
	// Upcoming is the list of songs that will play next.
	Upcoming []Song
	// Paused is true when the current song is paused.
	Paused bool
	// Mode is the current play mode.
	Mode PlayMode
	// Volume is the current volume level.
	Volume float64
	// Muted is true when the sound is muted.
	Muted bool
}

// Active reports whether a song is playing and not paused.
func (s Snapshot) Active() bool { return s.Now != nil && !s.Paused }

// PlaybackController connects the queue with the playback engine.
// It is safe for concurrent use.
type PlaybackController struct {
	mu     sync.Mutex
	engine PlaybackEngine
	queue  *Queue

	// generation identifies the current playback, so that
	// a "finished" callback of an old song is ignored.
	generation uint64
	version    uint64
	paused     bool
	closed     bool

	errs     []error
	onChange func(Snapshot)
	onError  func(error)
}

// NewPlaybackController creates a controller with an empty queue.
func NewPlaybackController(engine PlaybackEngine) *PlaybackController {
	return &PlaybackController{
		engine: engine,
		queue:  NewQueue(),
	}
}

// OnChange sets a callback that is called on every change of the controller state.
func (c *PlaybackController) OnChange(fn func(Snapshot)) {
	c.mu.Lock()
	c.onChange = fn
	c.mu.Unlock()
}

// OnError sets a callback that is called for every error that happens during playback.
func (c *PlaybackController) OnError(fn func(error)) {
	c.mu.Lock()
	c.onError = fn
	c.mu.Unlock()
}

// do runs fn under the lock. If fn reports a change, do increases the version
// and calls the OnChange callback. Collected errors are passed to the OnError callback.
// Callbacks are called after the lock is released.
// If the controller is closed, do does nothing.
func (c *PlaybackController) do(fn func() (changed bool)) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}

	changed := fn()

	var (
		snap     Snapshot
		onChange func(Snapshot)
		onError  func(error)
		errs     = c.errs
	)
	c.errs = nil
	if changed {
		c.version++
		snap = c.snapshot()
		onChange = c.onChange
	}
	onError = c.onError
	c.mu.Unlock()

	if onChange != nil {
		onChange(snap)
	}
	if onError != nil {
		for _, err := range errs {
			onError(err)
		}
	}
}

// snapshot builds a snapshot of the current state. The caller must hold the lock.
func (c *PlaybackController) snapshot() Snapshot {
	level, muted := c.engine.Volume()
	return Snapshot{
		Version:  c.version,
		Now:      c.queue.Now(),
		Upcoming: c.queue.Upcoming(),
		Paused:   c.paused,
		Mode:     c.queue.Mode(),
		Volume:   level,
		Muted:    muted,
	}
}

// Snapshot returns the current controller state.
func (c *PlaybackController) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshot()
}

// Mode returns the current play mode.
func (c *PlaybackController) Mode() PlayMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.queue.Mode()
}

// Shuffle shuffles the queue.
func (c *PlaybackController) Shuffle() {
	c.do(func() bool {
		c.queue.Shuffle()
		return true
	})
}

// Progress returns the position and length of the current song.
func (c *PlaybackController) Progress() (pos, length time.Duration) {
	return c.engine.Position()
}

// PlayFrom replaces the current queue with the given songs and starts playing.
func (c *PlaybackController) PlayFrom(songs []Song) {
	c.do(func() bool {
		c.queue.PlayFrom(songs)
		c.next()
		return true
	})
}

// PlayAt immediately starts playing the song at the given index.
// Skipped songs are saved in the history.
func (c *PlaybackController) PlayAt(idx int) {
	c.do(func() bool {
		song, ok := c.queue.JumpTo(idx)
		if !ok {
			return false
		}
		if !c.start(song) {
			c.next()
		}
		return true
	})
}

// Next switches to the next song.
func (c *PlaybackController) Next() {
	c.do(func() bool {
		c.next()
		return true
	})
}

// Previous goes back to the previous song in the history.
// If the history is empty, it restarts the current song.
func (c *PlaybackController) Previous() {
	c.do(func() bool {
		if song, ok := c.queue.Previous(); ok {
			if !c.start(song) {
				c.next()
			}
			return true
		}
		if now := c.queue.Now(); now != nil {
			c.start(now)
			return true
		}
		return false
	})
}

// Stop stops playback and finishes the queue.
func (c *PlaybackController) Stop() {
	c.do(func() bool {
		c.stop()
		return true
	})
}

// TogglePause pauses the current song or resumes it if it is paused.
// It does nothing if no song is loaded.
func (c *PlaybackController) TogglePause() {
	c.do(func() bool {
		if c.queue.Now() == nil {
			return false
		}
		c.engine.TogglePause()
		c.paused = !c.paused
		return true
	})
}

// Seek moves the position of the current song by delta (negative to go back).
// It does nothing if no song is loaded.
func (c *PlaybackController) Seek(delta time.Duration) {
	c.do(func() bool {
		if c.queue.Now() == nil {
			return false
		}
		if err := c.engine.Seek(delta); err != nil {
			c.errs = append(c.errs, fmt.Errorf("seek: %w", err))
			return false
		}
		return true
	})
}

// ChangeVolume changes the volume by delta.
func (c *PlaybackController) ChangeVolume(delta float64) {
	c.do(func() bool {
		c.engine.ChangeVolume(delta)
		return true
	})
}

// ToggleMute mutes the sound or unmutes it if it is muted.
func (c *PlaybackController) ToggleMute() {
	c.do(func() bool {
		c.engine.ToggleMute()
		return true
	})
}

// CycleMode switches the mode and returns the new mode.
func (c *PlaybackController) CycleMode() (mode PlayMode) {
	c.do(func() bool {
		mode = c.queue.Mode().Next()
		c.queue.SetMode(mode)
		return true
	})
	return mode
}

// Enqueue adds songs to the end of the queue. It does not start playback.
func (c *PlaybackController) Enqueue(songs ...Song) (added, skipped int) {
	c.do(func() bool {
		added, skipped = c.queue.Enqueue(songs...)
		return added > 0
	})
	return added, skipped
}

// Remove removes the songs with the given paths and returns how many were removed.
func (c *PlaybackController) Remove(paths ...string) (removed int) {
	c.do(func() bool {
		removed = c.queue.Remove(paths...)
		return removed > 0
	})
	return removed
}

// Move shifts the song at idx by delta positions and returns its new index.
func (c *PlaybackController) Move(idx, delta int) (newIdx int) {
	c.do(func() bool {
		newIdx = c.queue.Move(idx, delta)
		return newIdx != idx
	})
	return newIdx
}

// Clear empties the queue. The current song keeps playing.
func (c *PlaybackController) Clear() {
	c.do(func() bool {
		c.queue.Clear()
		return true
	})
}

// Close stops playback. After Close, all other methods do nothing.
func (c *PlaybackController) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.generation++
	c.engine.Stop()
}

// next switches to the next playable song.
// If a song fails to start, it is skipped.
// The number of attempts is at most the queue length plus one.
// If there is nothing to play, it stops playback.
func (c *PlaybackController) next() {
	for attempts := c.queue.Len() + 1; attempts > 0; attempts-- {
		song, ok := c.queue.Next()
		if !ok {
			c.stop()
			return
		}
		if c.start(song) {
			return
		}
	}
	c.stop()
}

// start starts playing the song and reports whether it succeeded.
// On failure the error is saved and passed to the OnError callback.
// The caller must hold the lock.
func (c *PlaybackController) start(song Song) bool {
	c.generation++
	gen := c.generation
	c.paused = false

	pErr := c.engine.Play(song, func() {
		go c.handleFinished(gen)
	})
	if pErr != nil {
		err := ErrorMessage{
			err:     pErr,
			errView: fmt.Errorf("[ %s ]\nFailed to play the song.\nSee the log file for details.\n", song.Name()),
		}
		c.errs = append(c.errs, err)
		return false
	}
	return true
}

// stop stops the engine and finishes the queue. The caller must hold the lock.
func (c *PlaybackController) stop() {
	c.generation++
	c.paused = false
	c.engine.Stop()
	c.queue.Finish()
}

// handleFinished is called when a song ends. It switches to the next song,
// unless the song has already been replaced (the generation has changed).
func (c *PlaybackController) handleFinished(gen uint64) {
	c.do(func() bool {
		if gen != c.generation {
			return false
		}
		c.next()
		return true
	})
}
