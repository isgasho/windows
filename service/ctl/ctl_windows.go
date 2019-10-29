package ctl

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func (s *Server) Start() error {
	ln, err := winio.ListenPipe(`\\.\pipe\`+s.Namespace, &winio.PipeConfig{
		SecurityDescriptor: "O:SYD:P(A;;GA;;;WD)",
	})
	if err != nil {
		return err
	}
	s.closer = ln
	go s.run(ln)
	if s.OnStart != nil {
		s.OnStart()
	}
	return nil
}

func (s *Server) run(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			s.logErr(err)
			continue
		}
		go s.handleEvents(c)
	}
}
