package main

import (
	// "log"
	// "time"
	//
	"os"
	// "fmt"
	"github.com/rivo/tview"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		exit(err)
	}
	dirEntries,err := os.ReadDir("/home/fioneo/Music/playlists/playlist3")
	if err != nil {
		exit(err)
	}

	playlistInfo := directoryToPlaylistInfo("playlist3","/home/fioneo/Music/playlists/playlist3",dirEntries)
	library := NewLibrary(homeDir)
	playlist := NewPlayPanel(playlistInfo)
	playbar := NewPlaybar()

	flex := tview.NewFlex().
		AddItem(library, 0, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(playlist, 0, 4, false).AddItem(playbar, 0, 2, false), 0, 3, false)

	if err := tview.NewApplication().SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
