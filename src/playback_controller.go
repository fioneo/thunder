package main

import (
	"fmt"
	"sync"
	"time"
)

// PlaybackEngine implements Player.
type PlaybackEngine interface {
	Play(song Song, onFinished func()) error
	Stop()
	TogglePause()
	Seek(delta time.Duration) error
	Position() (pos, length time.Duration)
	ChangeVolume(delta float64)
	ToggleMute()
	Volume() (level float64, muted bool)
}

// Snapshot is immutable state for the interface
type Snapshot struct {
	Version  uint64
	Now      Song
	Upcoming []Song
	Paused   bool
	Mode     PlayMode
	Volume   float64
	Muted    bool
}

// Active returns true whether current song is playing and it's not paused
func (s Snapshot) Active() bool { return s.Now != nil && !s.Paused }

func (s Snapshot) VolumePercent() int {
	return int(s.Volume*10) + 100
}

type PlaybackController struct {
	mu     sync.Mutex
	engine PlaybackEngine
	queue  *Queue

	generation uint64
	version    uint64
	paused     bool
	closed     bool

	errs     []error
	onChange func(Snapshot)
	onError  func(error)
}

func NewPlaybackController(engine PlaybackEngine) *PlaybackController {
	return &PlaybackController{
		engine: engine,
		queue:  NewQueue(),
	}
}

// OnChange takes callback and calls it on every change of the controller state.
func (c *PlaybackController) OnChange(fn func(Snapshot)) {
	c.mu.Lock()
	c.onChange = fn
	c.mu.Unlock()
}

func (c *PlaybackController) OnError(fn func(error)) {
	c.mu.Lock()
	c.onError = fn
	c.mu.Unlock()
}

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

// Shapshot return actual controller state.
func (c *PlaybackController) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshot()
}

func (c *PlaybackController) Mode() PlayMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.queue.Mode()
}

func (c *PlaybackController) Shuffle() {
	c.do(func() bool {
		c.queue.Shuffle()
		return true
	})
}

// Progress return position and length of the current song.
func (c *PlaybackController) Progress() (pos, length time.Duration) {
	return c.engine.Position()
}

// PlayFrom replace current queue with given songs
func (c *PlaybackController) PlayFrom(songs []Song) {
	c.do(func() bool {
		c.queue.PlayFrom(songs)
		c.next()
		return true
	})
}

// PlayAt immediatelly starts playing song with given index.
// Skipped songs are saved in history.
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

// Next goes to the next song.
func (c *PlaybackController) Next() {
	c.do(func() bool {
		c.next()
		return true
	})
}

// Previous goes to the previous song in history. If history is empty,next step depends on the mode.
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

func (c *PlaybackController) Stop() {
	c.do(func() bool {
		c.stop()
		return true
	})
}

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

func (c *PlaybackController) ChangeVolume(delta float64) {
	c.do(func() bool {
		c.engine.ChangeVolume(delta)
		return true
	})
}

func (c *PlaybackController) ToggleMute() {
	c.do(func() bool {
		c.engine.ToggleMute()
		return true
	})
}

// CycleMode cycling modes in order Seq -> Repeat -> Rand and returns new one
func (c *PlaybackController) CycleMode() (mode PlayMode) {
	c.do(func() bool {
		mode = c.queue.Mode().Next()
		c.queue.SetMode(mode)
		return true
	})
	return mode
}

// Enqueue adds song to the end of the queue. It will not start queue automatically.
func (c *PlaybackController) Enqueue(songs ...Song) (added, skipped int) {
	c.do(func() bool {
		added, skipped = c.queue.Enqueue(songs...)
		return added > 0
	})
	return added, skipped
}

// Remove removes songs with given path.
func (c *PlaybackController) Remove(paths ...string) (removed int) {
	c.do(func() bool {
		removed = c.queue.Remove(paths...)
		return removed > 0
	})
	return removed
}

// Move shifts song on delta position and returns it's new index
func (c *PlaybackController) Move(idx, delta int) (newIdx int) {
	c.do(func() bool {
		newIdx = c.queue.Move(idx, delta)
		return newIdx != idx
	})
	return newIdx
}

// Clear queue. Playing song continues to play.
func (c *PlaybackController) Clear() {
	c.do(func() bool {
		c.queue.Clear()
		return true
	})
}

// Close stops playing. After Close other methods do nothing.
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

// next goes to the next playable song.
// If there is an error,it skips the song.
// Number of attemps is less or equal to length of the queue.
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

func (c *PlaybackController) start(song Song) bool {
	c.generation++
	gen := c.generation
	c.paused = false

	err := c.engine.Play(song, func() {
		go c.handleFinished(gen)
	})
	if err != nil {
		c.errs = append(c.errs, err)
		return false
	}

	thunder.Logger.Info(fmt.Sprintf("Start playing song: %s", song.Name()))
	return true
}

func (c *PlaybackController) stop() {
	c.generation++
	c.paused = false
	c.engine.Stop()
	c.queue.Finish()
}

func (c *PlaybackController) handleFinished(gen uint64) {
	c.do(func() bool {
		if gen != c.generation {
			return false
		}
		c.next()
		return true
	})
}
