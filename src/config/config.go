package config

import (
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"io"
	"os"
	"strings"
)

type Global struct {
	VolumeStep float64 `json:"volume_step"`
	SeekStep   float64 `json:"seek_step"`
}

type Colors struct {
	Accent      tcell.Color `json:"accent"`
	Song        tcell.Color `json:"song"`
	Directory   tcell.Color `json:"directory"`
	Background  tcell.Color `json:"background"`
	Foreground  tcell.Color `json:"foreground"`
	Popup       tcell.Color `json:"popup"`
	Selected    tcell.Color `json:"selected"`
	Highlighted tcell.Color `json:"highlighted"`
	Copied      tcell.Color `json:"copied"`
	Placeholder tcell.Color `json:"placeholder"`
}

type Config struct {
	Global `json:"global"`
	Colors `json:"colors"`
}

func Parse(path string) (*Config, error) {
	cfg := DefaultConfig()
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(f).Decode(&cfg); err != nil && err != io.EOF {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	return cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Global: Global{
			VolumeStep: 0.5,
			SeekStep:   5,
		},
		Colors: DefaultColors(),
	}
}

func DefaultColors() Colors {
	return Colors{
		Accent:      tcell.GetColor("#6DB2EE"),
		Background:  tcell.ColorDefault,
		Foreground:  tcell.GetColor("#FEF9FF"),
		Song:        tcell.GetColor("#7CA7DF"),
		Directory:   tcell.GetColor("#6DB2EE"),
		Popup:       tcell.GetColor("#6DB2EE"),
		Selected:    tcell.GetColor("#9F9FED"),
		Highlighted: tcell.GetColor("#59BDEB"),
		Copied:      tcell.GetColor("#F3C98B"),
		Placeholder: tcell.GetColor("#595959"),
	}
}
