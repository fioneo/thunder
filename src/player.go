package main

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
	sampleRate = beep.SampleRate(48000)

	volumeStep = 0.5
	minVolume  = -10.0
	maxVolume  = 0.0
)

type audio struct {
	streamer beep.StreamSeekCloser
	format   beep.Format
	ctrl     *beep.Ctrl
	vol      *effects.Volume
	length   time.Duration
}

type Player struct {
	mu           sync.Mutex
	speakerReady bool
	audio        *audio
	volume       float64
	muted        bool
}

var _ PlaybackEngine = (*Player)(nil)

func NewPlayer(volume float64) *Player {
	return &Player{
		volume: volume,
	}
}

func (p *Player) Play(song Song, onFinished func()) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stop()

	f, err := os.Open(song.Path())
	if err != nil {
		return fmt.Errorf("[ %s ]\nOpen file: %w", song.Name(), err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("[ %s ]\nDecode song: %w", song.Name(), err)
	}

	if err := p.initSpeaker(); err != nil {
		streamer.Close()
		return fmt.Errorf("[ %s ]\nInitialize speaker: %w", song.Name(), err)
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

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stop()
}

func (p *Player) stop() {
	if p.audio == nil {
		return
	}
	speaker.Clear()
	p.audio.streamer.Close()
	p.audio = nil
}

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

func (p *Player) ToggleMute() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.muted = !p.muted
	p.applyVolume()
}

func (p *Player) Volume() (level float64, muted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume, p.muted
}

func (p *Player) applyVolume() {
	if p.audio == nil {
		return
	}
	speaker.Lock()
	defer speaker.Unlock()
	p.audio.vol.Volume = p.volume
	p.audio.vol.Silent = p.muted
}
