package svc

import (
	"fmt"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

type logger struct {
	elog debug.Log
}

func (l logger) Info(msg string) {
	l.elog.Info(1, msg)
}

func (l logger) Warn(msg string) {
	l.elog.Warning(1, msg)
}

func (l logger) Error(msg string) {
	l.elog.Error(1, msg)
}

type service struct {
	Service
	log Logger
}

func (s service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	if err := s.Start(s.log); err != nil {
		s.log.Error(fmt.Sprint(err))
		return true, 1
	}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			if err := s.Stop(s.log); err != nil {
				s.log.Error(fmt.Sprint(err))
				return true, 2
			}
			break loop
		}
	}

	return false, 0
}

func run(s Service, name string, isDebug bool) error {
	var elog debug.Log
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return err
		}
	}

	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	runner := svc.Run
	if isDebug {
		runner = debug.Run
	}
	err = runner(name, service{s, logger{elog}})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return err
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
	return nil
}
