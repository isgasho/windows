package ctl

import (
	"net"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/ipc/winpipe"
)

func (s *Server) Start() error {
	sec, err := windows.SecurityDescriptorFromString("O:SYD:P(A;;GA;;;WD)")
	if err != nil {
		return err
	}
	ln, err := winpipe.ListenPipe(`\\.\pipe\`+s.Namespace, &winpipe.PipeConfig{
		SecurityDescriptor: sec,
	})
	if err != nil {
		return err
	}
	s.closer = ln
	go s.run(ln)
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
