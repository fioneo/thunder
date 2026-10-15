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

func directoryToPlaylistInfo(name, path string, files []File) *playlistInfo {
	songs := make([]Song, 0, len(files))

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		song := fileToSong(f)

		songs = append(songs, song)
	}

	return NewPlaylistInfo(name, path, songs)
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

	// else if highlighted {
	// 		screen.SetContent(x, y, ' ', nil, highlightStyle)
	// 	}

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
			finalStyle := style
			// if maintainBackground {
			// 	_, currentBg, _ := style.Decompose()
			// 	if currentBg == tcell.ColorDefault {
			// 		_, _, existingStyle, _ := screen.GetContent(x, y)
			// 		_, screenBg, _ := existingStyle.Decompose()
			// 		finalStyle = finalStyle.Background(screenBg)
			// 	}
			// }

			screen.SetContent(x, y, r, nil, finalStyle)
		}
		x++
		printedWidth++
	}

	return printedWidth
}
