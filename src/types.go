package main

type PlayMode int

const (
	SequenceMode PlayMode = iota
	RepeatMode

	playModeCount // = 2
)

func (pm PlayMode) Next() PlayMode {
	return (pm + 1) % playModeCount // Cycling modes.
}

func (pm PlayMode) ToString() string {
	var str string

	switch pm {
	case SequenceMode:
		str = "Seq"
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
