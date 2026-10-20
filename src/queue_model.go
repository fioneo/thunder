package main

import (
	"math/rand"
	"slices"
)

// Queue is clear model which is not thread safe.It's used in PlaybackController
type Queue struct {
	now      Song
	upcoming []Song
	history  []Song
	mode     PlayMode
}

func NewQueue() *Queue {
	return &Queue{}
}

// Now returns current song or nil.
func (q *Queue) Now() Song { return q.now }

// Upcoming returns copy of the upcoming songs.
func (q *Queue) Upcoming() []Song { return slices.Clone(q.upcoming) }

// Len returns length of the upcoming songs.
func (q *Queue) Len() int { return len(q.upcoming) }

// Mode returns current playing mode.
func (q *Queue) Mode() PlayMode { return q.mode }

func (q *Queue) SetMode(mode PlayMode) {
	q.mode = mode
}

// TODO: Is it a good choise for shuffling?Maybe there is better algorithm
func (q *Queue) Shuffle() {
	rand.Shuffle(len(q.upcoming), func(i, j int) {
		q.upcoming[i], q.upcoming[j] = q.upcoming[j], q.upcoming[i]
	})
}

// Enqueue adds songs to the end of the queue, it skips already existing songs (including currently playing song).
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

func (q *Queue) PlayFrom(songs []Song) {
	q.Reset()
	q.Enqueue(songs...)
}

func (q *Queue) Next() (Song, bool) {
	// If mode is set to repeat,return current song
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

func (q *Queue) Previous() (Song, bool) {
	n := len(q.history)
	if n == 0 {
		return nil, false
	}

	prev := q.history[n-1]
	q.history = q.history[:n-1]

	q.upcoming = slices.DeleteFunc(q.upcoming, func(s Song) bool {
		return s.Path() == prev.Path()
	})
	if q.now != nil {
		q.upcoming = slices.Insert(q.upcoming, 0, q.now)
	}
	q.now = prev
	return prev, true
}

// Jumpto starts playing songs with given index. All previous songs are added to the history.
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

// Finish adds playing song to the history.
func (q *Queue) Finish() {
	if q.now != nil {
		q.history = append(q.history, q.now)
		q.now = nil
	}
}

func (q *Queue) Remove(paths ...string) int {
	drop := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		drop[p] = struct{}{}
	}

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

// Move shifts element to the given index (idx + delta).
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

func (q *Queue) Clear() {
	clear(q.upcoming)
	q.upcoming = q.upcoming[:0]
}

func (q *Queue) Reset() {
	q.now = nil
	q.upcoming = nil
	q.history = nil
}
