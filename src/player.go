package main

import (
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"os"
	"time"
)

type Player struct {
	hasInit     bool
	currentSong Song
	isRunning   bool
	volume      float64
	mode        PlayMode
	length      time.Duration

	vol      *effects.Volume
	ctrl     *beep.Ctrl
	format   *beep.Format
	streamer beep.StreamSeekCloser

	// mu sync.Mutex
}

func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) Stream(song Song) error {
	f, err := os.Open(song.Path())
	if err != nil {
		return fmtError("open file", err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return fmtError("decode song", err)
	}
	defer streamer.Close()

	p.length = format.SampleRate.D(streamer.Len())
	p.mode = DefaultMode

	p.streamer = streamer
	p.format = &format

	p.currentSong = song
	p.isRunning = true

	sr := beep.SampleRate(48000)

	if !p.hasInit {
		if err := speaker.Init(sr, format.SampleRate.N(time.Second/10)); err != nil {
			return fmtError("init speaker", err)
		}
		p.hasInit = true
	}

	resampled := beep.Resample(4, format.SampleRate, sr, streamer)

	resampledStreamer := beep.Seq(resampled, beep.Callback(func() {
		p.isRunning = false
		p.format = nil
		if p.streamer != nil {
			p.streamer.Close()
		}
	}))

	ctrl := &beep.Ctrl{
		Streamer: resampledStreamer,
		Paused:   false,
	}

	resampler := beep.ResampleRatio(4, 1, ctrl)

	vol := &effects.Volume{
		Streamer: resampler,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}

	p.ctrl = ctrl
	p.vol = vol

	p.volume = vol.Volume

	speaker.Play(vol)

	return nil
}

func (p *Player) TogglePause() {
	speaker.Lock()
	defer speaker.Unlock()
	if p.ctrl != nil {
		p.ctrl.Paused = !p.ctrl.Paused
	}
	p.isRunning = !p.isRunning
}

func (p *Player) Skip() {}

func (p *Player) Seek(n time.Duration) error {
	speaker.Lock()
	defer speaker.Unlock()
	if p.streamer == nil {
		return nil
	}

	newPos := p.streamer.Position() + p.format.SampleRate.N(n)

	newPos = max(newPos, 0)
	length := p.streamer.Len()
	if length >= 1 {
		length = length - 1
	}
	newPos = min(newPos, length)

	return p.streamer.Seek(newPos)
}

func (p *Player) ChangeMode() {
	speaker.Lock()
	speaker.Unlock()
}

func (p *Player) LowerVolume() {
	speaker.Lock()
	if p.vol == nil {
		return
	}
	p.vol.Volume -= 0.5
	p.volume = p.vol.Volume
	speaker.Unlock()
}

func (p *Player) HigherVolume() {
	speaker.Lock()
	if p.vol == nil {
		return
	}
	p.vol.Volume += 0.5
	p.volume = p.vol.Volume
	speaker.Unlock()
}

func (p *Player) ToggleMute() {
	speaker.Lock()
	if p.vol == nil {
		return
	}
	p.vol.Silent = !p.vol.Silent
	speaker.Unlock()
}

func (p *Player) SpeedUp() {
	speaker.Lock()
	if p.vol == nil {
		return
	}
	ratio := p.vol.Streamer.(*beep.Resampler).Ratio()
	if ratio >= 3 {
		return
	}
	resampler := beep.ResampleRatio(4, ratio+0.5, p.ctrl)
	p.vol.Streamer = resampler
	speaker.Unlock()
}

func (p *Player) SpeedDown() {
	speaker.Lock()
	if p.vol == nil {
		return
	}
	ratio := p.vol.Streamer.(*beep.Resampler).Ratio()
	if ratio <= 1 {
		return
	}
	resampler := beep.ResampleRatio(4, ratio-0.5, p.ctrl)
	p.vol.Streamer = resampler
	speaker.Unlock()
}

func (p *Player) CurrentVolume() float64 {
	return p.volume
}

func (p *Player) CurrentSong() Song {
	return p.currentSong
}

func (p *Player) Length() time.Duration {
	return p.length
}

func (p *Player) CurrentPosition() time.Duration {
	speaker.Lock()
	defer speaker.Unlock()
	if p.format == nil || p.streamer == nil {
		return 0
	}
	return p.format.SampleRate.D(p.streamer.Position())
}

func (p *Player) IsRunning() bool {
	return p.isRunning
}

func (p *Player) IsPaused() bool {
	if p.ctrl != nil {
		return p.ctrl.Paused
	}
	return false
}
