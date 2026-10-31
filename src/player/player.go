package player

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
)

const (
	// sampleRate is the sample rate of the speaker. Songs are resampled to it.
	sampleRate = beep.SampleRate(48000)

	// Volume is a power of 2 (effects.Volume with Base 2): 0 is the original
	// loudness, negative values are quieter.
	volumeStep = 0.5
	minVolume  = -10.0
	maxVolume  = 0.0
)

// audio holds everything related to the currently loaded song.
type audio struct {
	streamer beep.StreamSeekCloser
	format   beep.Format
	ctrl     *beep.Ctrl
	vol      *effects.Volume
	length   time.Duration
}

// Player plays mp3 files through the speaker. It is safe for concurrent use.
type Player struct {
	mu sync.Mutex
	// speakerReady is true after the speaker has been initialized.
	speakerReady bool
	// audio is the current song, or nil if nothing is loaded.
	audio *audio
	// volume and muted are kept here, so they are applied to every new song.
	volume float64
	muted  bool
}

var _ PlaybackEngine = (*Player)(nil)

// NewPlayer creates a player with the given initial volume.
func NewPlayer(volume float64) *Player {
	return &Player{
		volume: volume,
	}
}

// Play stops the current song and starts the given one.
// onFinished is called when the song ends on its own (not when it is stopped).
func (p *Player) Play(song Song, onFinished func()) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stop()

	f, err := os.Open(song.Path())
	if err != nil {
		return fmt.Errorf("[ %s ] Open file: %w", song.Name(), err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("[ %s ] Decode song: %w", song.Name(), err)
	}

	if err := p.initSpeaker(); err != nil {
		streamer.Close()
		return fmt.Errorf("[ %s ] Initialize speaker: %w", song.Name(), err)
	}

	ctrl := &beep.Ctrl{Streamer: beep.Resample(4, format.SampleRate, sampleRate, streamer)}
	vol := &effects.Volume{Streamer: ctrl, Base: 2, Volume: p.volume, Silent: p.muted}

	p.audio = &audio{
		streamer: streamer,
		format:   format,
		ctrl:     ctrl,
		vol:      vol,
		length:   format.SampleRate.D(streamer.Len()),
	}

	if onFinished == nil {
		onFinished = func() {}
	}
	speaker.Play(beep.Seq(vol, beep.Callback(onFinished)))
	return nil
}

// initSpeaker initializes the speaker once. The caller must hold the lock.
func (p *Player) initSpeaker() error {
	if p.speakerReady {
		return nil
	}
	if err := speaker.Init(sampleRate, sampleRate.N(time.Second/10)); err != nil {
		return err
	}
	p.speakerReady = true
	return nil
}

// Stop stops the current song. It does nothing if nothing is playing.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stop()
}

// stop clears the speaker and closes the current song. The caller must hold the lock.
func (p *Player) stop() {
	if p.audio == nil {
		return
	}
	speaker.Clear()
	p.audio.streamer.Close()
	p.audio = nil
}

// TogglePause pauses the current song or resumes it if it is paused.
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.audio == nil {
		return
	}

	speaker.Lock()
	defer speaker.Unlock()
	p.audio.ctrl.Paused = !p.audio.ctrl.Paused
}

// Seek moves the position of the current song by delta (negative to go back).
// The new position is limited to the bounds of the song.
func (p *Player) Seek(delta time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.audio == nil {
		return nil
	}

	speaker.Lock()
	defer speaker.Unlock()

	t := p.audio
	pos := t.streamer.Position() + t.format.SampleRate.N(delta)
	last := max(t.streamer.Len()-1, 0)

	err := t.streamer.Seek(min(max(pos, 0), last))

	return err
}

// Position returns the current position and the length of the song.
// It returns zeros if nothing is loaded.
func (p *Player) Position() (pos, length time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.audio == nil {
		return 0, 0
	}

	speaker.Lock()
	defer speaker.Unlock()
	return p.audio.format.SampleRate.D(p.audio.streamer.Position()), p.audio.length
}

// ChangeVolume changes the volume by delta, limited to [minVolume, maxVolume].
// The sound is muted at the minimum volume and unmuted otherwise.
func (p *Player) ChangeVolume(delta float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.volume = min(max(p.volume+delta, minVolume), maxVolume)
	if p.volume == minVolume {
		p.muted = true
	} else {
		p.muted = false
	}
	p.applyVolume()
}

// ToggleMute mutes the sound or unmutes it if it is muted.
// The volume level is kept.
func (p *Player) ToggleMute() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.muted = !p.muted
	p.applyVolume()
}

// Volume returns the current volume level and whether it is muted.
func (p *Player) Volume() (level float64, muted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume, p.muted
}

// applyVolume applies the volume and mute state to the current song.
// The caller must hold the lock.
func (p *Player) applyVolume() {
	if p.audio == nil {
		return
	}
	speaker.Lock()
	defer speaker.Unlock()
	p.audio.vol.Volume = p.volume
	p.audio.vol.Silent = p.muted
}
