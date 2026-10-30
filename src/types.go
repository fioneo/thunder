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

// ErrorMessage wraps an error that is too big to show on the screen as is,
// because calculating its width and height is not trivial.
// err is written to the log file, and errView is a short description
// of what happened, which is shown on the screen.
type ErrorMessage struct {
	err     error
	errView error
}

// Error is never called. It exists only so that ErrorMessage
// implements the error interface.
func (m ErrorMessage) Error() string {
	return ""
}
