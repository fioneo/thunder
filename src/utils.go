package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"os"
)

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func fmtError(status string, err error) error {
	return fmt.Errorf("%s: %w", status, err)
}

func fileToSong(file File) Song {
	return NewSong(file.Name(), file.Path())
}

func directoryToPlaylistInfo(name, path string, files []File) playlistInfo {
	songs := make([]Song, 0, len(files))

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		song := NewSong(file.Name(), file.Path())

		songs = append(songs, song)
	}

	return NewPlaylistInfo(name, path, songs)
}

func printWithStyle(screen tcell.Screen, text string, x, y, skipWidth, maxWidth int, style tcell.Style, maintainBackground bool) (start, end, printedWidth int) {
	totalWidth, totalHeight := screen.Size()
	if maxWidth <= 0 || len(text) == 0 || y < 0 || y >= totalHeight {
		return 0, 0, 0
	}

	runes := []rune(text)

	if skipWidth > 0 {
		if skipWidth >= len(runes) {
			return len(runes), len(runes), 0
		}
		start = skipWidth
		runes = runes[skipWidth:]
	}

	if len(runes) > maxWidth {
		runes = runes[:maxWidth]
	}

	rightBorder := x + len(runes)

	for _, r := range runes {
		if x >= rightBorder || x >= totalWidth {
			break
		}

		if x >= 0 {
			finalStyle := style
			if maintainBackground {
				_, currentBg, _ := style.Decompose()
				if currentBg == tcell.ColorDefault {
					_, _, existingStyle, _ := screen.GetContent(x, y)
					_, screenBg, _ := existingStyle.Decompose()
					finalStyle = finalStyle.Background(screenBg)
				}
			}

			screen.SetContent(x, y, r, nil, finalStyle)
		}
		x++
		printedWidth++
	}

	end = start + printedWidth
	return start, end, printedWidth
}
