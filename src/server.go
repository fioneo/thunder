package main

import (
	"io"
	"net"
	"os"
	"sync"
)

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
	var song string
	if snap.Now != nil {
		song = snap.Now.Name()
	}
	line := []byte(song + "\n")

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

	go func() {
		for line := range ch {
			if _, err := conn.Write(line); err != nil {
				conn.Close()
				return
			}
		}
	}()

	s.Write(s.ctrl.Snapshot())

	_, _ = io.Copy(io.Discard, conn)
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
