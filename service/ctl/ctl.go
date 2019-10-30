package ctl

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// Server provides a bi-directional event stream with clients on top of named
// pipes.
type Server struct {
	Namespace string

	Handler EventHandler

	OnStart      func()
	OnConnect    func(c net.Conn)
	OnDisconnect func(c net.Conn)

	// ErrorLog specifies an optional log function for errors. If not set,
	// errors are not reported.
	ErrorLog func(error)

	mu      sync.Mutex
	clients []net.Conn
	closer  io.Closer
}

// Event represents an event either received from or sent to a client.
type Event struct {
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}

// EventHandler handles received events.
type EventHandler interface {
	HandleEvent(e Event)
}

type EventHandlerFunc func(e Event)

func (h EventHandlerFunc) HandleEvent(e Event) {
	h(e)
}

// Broadcast broadcasts e to all connected clients.
func (s *Server) Broadcast(e Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clients {
		_, err = c.Write(b)
		if err != nil {
			s.logErr(fmt.Errorf("write event: %v", err))
		}
	}
	return nil
}

func (s *Server) handleEvents(c net.Conn) {
	if s.OnConnect != nil {
		s.OnConnect(c)
	}
	s.addClient(c)
	defer func() {
		s.removeClient(c)
		c.Close()
		if s.OnDisconnect != nil {
			s.OnDisconnect(c)
		}
	}()
	dec := json.NewDecoder(c)
	for {
		var e Event
		err := dec.Decode(&e)
		if err != nil {
			if err != io.EOF {
				s.logErr(fmt.Errorf("decode event: %v", err))
			}
			break
		}
		if s.Handler != nil {
			go s.Handler.HandleEvent(e)
		}
	}
}

func (s *Server) addClient(c net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = append(s.clients, c)
}

func (s *Server) removeClient(c net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	clients := make([]net.Conn, 0, len(s.clients))
	for _, _c := range s.clients {
		if c == _c {
			continue
		}
		clients = append(s.clients, _c)
	}
	s.clients = clients
}

func (s *Server) logErr(err error) {
	if s.ErrorLog != nil {
		s.ErrorLog(err)
	}
}

// Stop stops listening on the named pipe.
func (s *Server) Stop() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = nil
	if s.closer != nil {
		err = s.closer.Close()
		s.closer = nil
	}
	return
}
