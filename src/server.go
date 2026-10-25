package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
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
	// Volume set in thunder in percent format ( n% ).
	Volume int `json:"volume"`
	// Muted is true when song is muted.
	Muted bool `json:"muted"`
	// Playing is true whether the current song is playing and it's not paused.
	Playing bool `json:"playing"`
	// Paused is here to distinguish between nothing is playing and the current song is paused.
	Paused bool `json:"paused"`
}

type Server struct {
	ctrl      *PlaybackController
	l         net.Listener
	path      string
	mu        sync.Mutex
	clients   map[net.Conn]chan []byte
	done      chan struct{}
	closeOnce sync.Once
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
		done:    make(chan struct{}),
	}
	thunder.Logger.Infof("Successfully initialized server on %s. Ready to accept connections", path)
	go s.loop()

	return s, nil
}

func (s *Server) loop() {
	var delay time.Duration
	for {
		conn, err := s.l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if delay == 0 {
				delay = 5 * time.Millisecond
			} else if delay *= 2; delay > time.Second {
				delay = time.Second
			}
			thunder.Logger.Errorf("accept connection: %s; retry in %s", err, delay)

			select {
			case <-time.After(delay):
			case <-s.done:
				return
			}
			continue
		}
		delay = 0
		go s.serve(conn)
		go s.read(conn)
	}
}

func (s *Server) Write(snap Snapshot) {
	pos, length := thunder.Playback.Progress()
	r := ResponseFromSnapshot(snap, pos, length)
	line, err := json.Marshal(r)
	if err != nil {
		thunder.Logger.Errorf("marshal json: %s", err)
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

	for {
		select {
		case line := <-ch:
			if _, err := conn.Write(line); err != nil {
				if err != io.EOF {
					thunder.Logger.Errorf("write to the client: %s", err)
				}
				return
			}
		case <-s.done:
			return
		}
	}
}

func (s *Server) read(conn net.Conn) {
	r := bufio.NewReader(conn)

	buff := make([]byte, 64)

	for {
		n, err := r.Read(buff)
		if n > 0 {
			cmd := strings.TrimSpace(string(buff[:n]))
			thunder.Logger.Infof("received from the client %s", cmd)
			switch cmd {
			case "TOGGLE_PAUSE":
				s.ctrl.TogglePause()
			case "NEXT":
				s.ctrl.Next()
			case "PREV":
				s.ctrl.Previous()
			case "TOGGLE_MUTE":
				s.ctrl.ToggleMute()
			case "VOL_UP":
				s.ctrl.ChangeVolume(volumeStep)
			case "VOL_DOWN":
				s.ctrl.ChangeVolume(-volumeStep)
			case "SEEK_BACKWARD":
				s.ctrl.Seek(seekStep)
			case "SEEK_FORWARD":
				s.ctrl.Seek(-seekStep)
			case "CYCLE_MODE":
				s.ctrl.CycleMode()
			}
		}
		if err != nil {
			if err != io.EOF {
				thunder.Logger.Errorf("read from the client: %v", err)
			}
			return
		}
	}
}

func (s *Server) Close() {
	s.closeOnce.Do(
		func() {
			s.l.Close()
			close(s.done)
			s.mu.Lock()
			defer s.mu.Unlock()
			for conn := range s.clients {
				conn.Close()
			}
		},
	)
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
		Volume:   s.VolumePercent(),
		Muted:    s.Muted,
		Playing:  s.Active(),
		Paused:   s.Paused,
	}
}
