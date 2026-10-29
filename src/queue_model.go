package main

import (
	"math/rand"
	"slices"
)

// Queue is a plain model of the play queue: the current song, the upcoming songs
// and the history of played songs. It is not safe for concurrent use;
// PlaybackController guards it with a mutex.
type Queue struct {
	now      Song
	upcoming []Song
	history  []Song
	mode     PlayMode
}

// NewQueue creates an empty queue.
func NewQueue() *Queue {
	return &Queue{}
}

// Now returns the current song, or nil if there is none.
func (q *Queue) Now() Song { return q.now }

// Upcoming returns a copy of the upcoming songs.
func (q *Queue) Upcoming() []Song { return slices.Clone(q.upcoming) }

// Len returns the number of upcoming songs.
func (q *Queue) Len() int { return len(q.upcoming) }

// Mode returns the current play mode.
func (q *Queue) Mode() PlayMode { return q.mode }

// SetMode sets the play mode.
func (q *Queue) SetMode(mode PlayMode) {
	q.mode = mode
}

// Shuffle randomly reorders the upcoming songs.
// The current song and the history are not affected.
func (q *Queue) Shuffle() {
	rand.Shuffle(len(q.upcoming), func(i, j int) {
		q.upcoming[i], q.upcoming[j] = q.upcoming[j], q.upcoming[i]
	})
}

// Enqueue adds songs to the end of the queue.
// It skips nil songs and songs that are already in the queue,
// including the current one. It returns the number of added and skipped songs.
func (q *Queue) Enqueue(songs ...Song) (added, skipped int) {
	seen := make(map[string]struct{}, len(q.upcoming)+1)
	if q.now != nil {
		seen[q.now.Path()] = struct{}{}
	}
	for _, s := range q.upcoming {
		seen[s.Path()] = struct{}{}
	}

	for _, s := range songs {
		if s == nil {
			skipped++
			continue
		}
		if _, dup := seen[s.Path()]; dup {
			skipped++
			continue
		}
		seen[s.Path()] = struct{}{}
		q.upcoming = append(q.upcoming, s)
		added++
	}
	return added, skipped
}

// PlayFrom replaces everything in the queue with the given songs.
// It does not select the current song; call Next for that.
func (q *Queue) PlayFrom(songs []Song) {
	q.Reset()
	q.Enqueue(songs...)
}

// Next moves to the next song and returns it.
// In repeat mode it returns the current song again.
// Otherwise the current song goes to the history. When no upcoming songs are
// left, the history becomes the upcoming songs (the queue starts over).
// It returns false if there is nothing to play.
func (q *Queue) Next() (Song, bool) {
	// In repeat mode, keep playing the current song.
	if q.mode == RepeatMode {
		if q.now != nil {
			return q.now, true
		}
	}
	q.Finish()

	if len(q.upcoming) == 0 {
		q.upcoming, q.history = q.history, nil
	}
	if len(q.upcoming) == 0 {
		return nil, false
	}

	q.now, q.upcoming = q.upcoming[0], q.upcoming[1:]
	return q.now, true
}

// Previous goes back to the last song in the history and returns it.
// The current song is put at the start of the upcoming songs.
// It returns false if the history is empty.
func (q *Queue) Previous() (Song, bool) {
	n := len(q.history)
	if n == 0 {
		return nil, false
	}

	prev := q.history[n-1]
	q.history = q.history[:n-1]

	// Make sure the song does not appear twice in the queue.
	q.upcoming = slices.DeleteFunc(q.upcoming, func(s Song) bool {
		return s.Path() == prev.Path()
	})
	if q.now != nil {
		q.upcoming = slices.Insert(q.upcoming, 0, q.now)
	}
	q.now = prev
	return prev, true
}

// JumpTo makes the upcoming song at idx the current one and returns it.
// The previous current song and all upcoming songs before idx are added
// to the history. It returns false if idx is out of range.
func (q *Queue) JumpTo(idx int) (Song, bool) {
	if idx < 0 || idx >= len(q.upcoming) {
		return nil, false
	}

	q.Finish()
	q.history = append(q.history, q.upcoming[:idx]...)
	q.now = q.upcoming[idx]
	q.upcoming = q.upcoming[idx+1:]
	return q.now, true
}

// Finish moves the current song to the history and leaves no current song.
func (q *Queue) Finish() {
	if q.now != nil {
		q.history = append(q.history, q.now)
		q.now = nil
	}
}

// Remove removes the upcoming songs with the given paths
// and returns how many were removed.
// The current song and the history are not affected.
func (q *Queue) Remove(paths ...string) int {
	drop := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		drop[p] = struct{}{}
	}

	// Filter in place, then clear the tail so removed songs can be garbage collected.
	kept := q.upcoming[:0]
	removed := 0
	for _, s := range q.upcoming {
		if _, ok := drop[s.Path()]; ok {
			removed++
			continue
		}
		kept = append(kept, s)
	}
	clear(q.upcoming[len(kept):])
	q.upcoming = kept
	return removed
}

// Move shifts the upcoming song at idx by delta positions and returns
// its new index. If either index is out of range, nothing changes
// and idx is returned.
func (q *Queue) Move(idx, delta int) int {
	dst := idx + delta
	if idx < 0 || idx >= len(q.upcoming) || dst < 0 || dst >= len(q.upcoming) {
		return idx
	}

	song := q.upcoming[idx]
	q.upcoming = slices.Delete(q.upcoming, idx, idx+1)
	q.upcoming = slices.Insert(q.upcoming, dst, song)
	return dst
}

// Clear removes all upcoming songs.
// The current song and the history are not affected.
func (q *Queue) Clear() {
	clear(q.upcoming)
	q.upcoming = q.upcoming[:0]
}

// Reset removes everything: the current song, the upcoming songs and the history.
func (q *Queue) Reset() {
	q.now = nil
	q.upcoming = nil
	q.history = nil
}
