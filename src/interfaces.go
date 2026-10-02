package main

type Song interface {
	Name() string
	Path() string
}

type PlaylistInfo interface {
	Name() string
	Path() string
	Songs() []Song
}
