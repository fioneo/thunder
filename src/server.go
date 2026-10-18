package main

import (
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"
)

type Response struct {
	// Name of the song.
	Name string `json:"name"`
	// Current position. Formats as m:ss or h:mm:ss.
	Position string `json:"position"`
	// Length of the song. Formats as m:ss or h:mm:ss.
	Length string `json:"length"`
	// Playing is true whether the current song is playing and it's not paused.
	Playing bool `json:"playing"`
	// Paused is here to distinguish between nothing is playing and the current song is paused.
	Paused bool `json:"paused"`
}

type Server struct {
	ctrl    *PlaybackController
	l       net.Listener
	path    string
	mu      sync.Mutex
	clients map[net.Conn]chan []byte
}

func NewServer(path string, ctrl *PlaybackController) (*Server, error) {
	_ = os.Remove(path)

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(path, 0o600); err != nil {
		l.Close()
		return nil, err
	}

	s := &Server{
		ctrl:    ctrl,
		l:       l,
		path:    path,
		clients: make(map[net.Conn]chan []byte),
	}

	go s.loop()

	return s, nil
}

func (s *Server) loop() {
	for {
		conn, err := s.l.Accept()
		if err != nil {
			return
		}
		go s.serve(conn)
	}
}

func (s *Server) Write(snap Snapshot) {

	pos, length := thunder.Playback.Progress()
	r := ResponseFromSnapshot(snap, pos, length)
	line, err := json.Marshal(r)
	if err != nil {
		return
	}

	line = append(line, byte('\n'))

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ch := range s.clients {
		select {
		case ch <- line:
		default:
		}
	}
}

func (s *Server) serve(conn net.Conn) {
	ch := make(chan []byte, 4)

	s.mu.Lock()
	s.clients[conn] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		close(ch)
		conn.Close()
	}()

	s.Write(s.ctrl.Snapshot())

	for line := range ch {
		if _, err := conn.Write(line); err != nil {
			conn.Close()
			return
		}
	}
}

func (s *Server) Close() {
	s.l.Close()

	s.mu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.mu.Unlock()

	_ = os.Remove(s.path)
}

func ResponseFromSnapshot(s Snapshot, pos, length time.Duration) Response {
	name := ""
	if s.Now != nil {
		name = s.Now.Name()
	}
	return Response{
		Name:     name,
		Position: fmtDuration(pos),
		Length:   fmtDuration(length),
		Playing:  s.Active(),
		Paused:   s.Paused,
	}
}
