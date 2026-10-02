package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func fmtError(status string, err error) error {
	return fmt.Errorf("%s: %w", status, err)
}

func fileInfoToSong(name,path string) Song {
	return NewSong(name,path)
}

func directoryToPlaylistInfo(name,path string,dirEntries []os.DirEntry) playlistInfo {
	songs := make([]Song,0,len(dirEntries))

	for _,entry := range dirEntries {
		song := NewSong(entry.Name(),filepath.Join(path,entry.Name()))

		songs = append(songs,song)
	}

	return NewPlaylistInfo(name,path,songs)
}
