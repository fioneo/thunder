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
		str = "Random"
	case RepeatMode:
		str = "Repeat"
	}

	return str
}

type song struct {
	name string
	path string
}

func NewSong(name, path string) song {
	return song{
		name: name,
		path: path,
	}
}

func (s song) Name() string {
	return s.name
}

func (s song) Path() string {
	return s.path
}

type playlistInfo struct {
	name  string
	path  string
	songs []Song
}

func NewPlaylistInfo(name string, path string, songs []Song) playlistInfo {
	return playlistInfo{
		name:  name,
		path:  path,
		songs: songs,
	}
}

func NewDefaultPlaylistInfo() playlistInfo {
	return NewPlaylistInfo("New Playlist", "", []Song{})
}

func (pi playlistInfo) Name() string {
	return pi.name
}

func (pi playlistInfo) Path() string {
	return pi.path
}

func (pi playlistInfo) Songs() []Song {
	return pi.songs
}

type node struct {
	isDir    bool
	children []Node
	name     string
	path     string
	parent   Node
}

func NewNode() *node {
	return &node{}
}

func (n *node) IsDir() bool {
	return n.isDir
}

func (n *node) Children() []Node {
	return n.children
}

func (n *node) Name() string {
	return n.name
}

func (n *node) Path() string {
	return n.path
}

func (n *node) Parent() Node {
	return n.parent
}

func (n *node) SetIsDir(isDir bool) {
	n.isDir = isDir
}

func (n *node) SetChildren(children []Node) {
	n.children = children
}

func (n *node) SetName(name string) {
	n.name = name
}

func (n *node) SetPath(path string) {
	n.path = path
}

func (n *node) SetParent(parent Node) {
	n.parent = parent
}
