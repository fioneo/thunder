package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func fileToSong(file File) Song {
	return NewSong(file.Name(), file.Path())
}

func filesToSongs(files []File) []Song {
	songs := make([]Song, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		song := fileToSong(f)

		songs = append(songs, song)
	}
	return songs
}

func printListItem(screen tcell.Screen, text string, x, y, skipWidth, maxWidth int, style tcell.Style, selectedStyle tcell.Style, selected bool, highlightStyle tcell.Style, highlighted bool) (printedWidth int) {
	totalWidth, totalHeight := screen.Size()
	if maxWidth <= 0 || len(text) == 0 || y < 0 || y >= totalHeight {
		return 0
	}
	runes := []rune(text)

	if skipWidth > 0 {
		if skipWidth >= len(runes) {
			return 0
		}
		runes = runes[skipWidth:]
	}

	if len(runes) > maxWidth {
		runes = runes[:maxWidth]
	}

	if selected {
		screen.SetContent(x, y, ' ', nil, selectedStyle)
	} else {
		screen.SetContent(x, y, ' ', nil, tcell.StyleDefault.Background(thunder.Colors.Background))
	}

	x++
	screen.SetContent(x, y, ' ', nil, tcell.StyleDefault.Background(thunder.Colors.Background))
	x++
	printedWidth += 2

	rightBorder := x + len(runes)

	for _, r := range runes {
		if x >= rightBorder || x >= totalWidth {
			break
		}

		if x >= 0 {

			screen.SetContent(x, y, r, nil, style)
		}
		x++
		printedWidth++
	}

	return printedWidth
}

func PrintText(screen tcell.Screen, x, y, maxWidth int, text string, style tcell.Style) (printed int) {
	for _, r := range text {
		if printed >= maxWidth {
			break
		}
		screen.SetContent(x+printed, y, r, nil, style)
		printed++
	}
	return printed
}

// fmtDuration formats time duration as m:ss or h:mm:ss
func fmtDuration(d time.Duration) string {
	total := int(d.Round(time.Second).Seconds())
	if total < 0 {
		total = 0
	}
	h, m, s := total/3600, total/60%60, total%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func percentageToVolume(percentage int) float64 {
	return float64(percentage-100) / 10
}

func volumeToPercentage(volume float64) int {
	return int(volume*10) + 100
}

func songWord(n int) string {
	if n == 1 {
		return "song"
	}
	return "songs"
}

func SendErr(err error) {
	if err == nil {
		return
	}
	select {
	case thunder.ErrCh <- err:
	default:
	}
}

func onUI(f func()) {
	thunder.App.QueueUpdateDraw(f)
}
