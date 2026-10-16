package main

type PlayMode int

const (
	DefaultMode PlayMode = iota
	RepeatMode
	RandomMode
)

func (pm PlayMode) ToString() string {
	var str string

	switch pm {
	case DefaultMode:
		str = "Seq"
	case RandomMode:
		str = "Rand"
	case RepeatMode:
		str = "Repeat"
	}

	return str
}

type song struct {
	name string
	path string
	// length time.Duration
}

func NewSong(name, path string) *song {
	return &song{
		name: name,
		path: path,
		// length: length,
	}
}

func (s *song) Name() string {
	return s.name
}

func (s *song) Path() string {
	return s.path
}

// func (s *song) Length() time.Duration {
// 	return s.length
// }

type playlistInfo struct {
	name  string
	path  string
	songs []Song
}

func NewPlaylistInfo(name string, path string, songs []Song) *playlistInfo {
	return &playlistInfo{
		name:  name,
		path:  path,
		songs: songs,
	}
}

func NewDefaultPlaylistInfo(path string) *playlistInfo {
	return NewPlaylistInfo("New Playlist", path, []Song{})
}

func (pi *playlistInfo) Name() string {
	return pi.name
}

func (pi *playlistInfo) Path() string {
	return pi.path
}

func (pi *playlistInfo) Songs() []Song {
	return pi.songs
}

type file struct {
	isDir bool
	name  string
	path  string
}

func NewFile(name, path string, isDir bool) *file {
	return &file{
		isDir: isDir,
		name:  name,
		path:  path,
	}
}

func (f *file) IsDir() bool {
	return f.isDir
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Path() string {
	return f.path
}

type Set struct {
	items map[string]struct{}
}

func NewSet() *Set {
	return &Set{
		items: make(map[string]struct{}),
	}
}

func (s *Set) Add(path string) {
	s.items[path] = struct{}{}
}

func (s *Set) Remove(path string) {
	delete(s.items, path)
}

func (s *Set) Exist(path string) bool {
	_, ok := s.items[path]

	return ok
}

func (s *Set) Grow(capacity int) {
	s.items = make(map[string]struct{}, capacity)
}

func (s *Set) Clear() {
	s.items = make(map[string]struct{})
}

