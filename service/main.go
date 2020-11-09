package main

import (
	"flag"
	"fmt"
	"hash/crc64"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/denisbrodbeck/machineid"

	"github.com/nextdns/windows/ctl"
	"github.com/nextdns/windows/proxy"
	"github.com/nextdns/windows/settings"
	"github.com/nextdns/windows/svc"
	"github.com/nextdns/windows/updater"
	"github.com/nextdns/windows/windoh"
)

type impl interface {
	SetConfigID(id string)
	SetDeviceInfo(name, model, id, version string)
	State() string
	Start() error
	Stop() error
}

type nextdnsSvc struct {
	impl impl
	ctl  ctl.Server
	log  svc.Logger
}

func (s *nextdnsSvc) Start(log svc.Logger) error {
	s.log = log
	log.Info("Service starting")
	defer log.Info("Service started")
	return s.ctl.Start()
}

func (s *nextdnsSvc) Stop(log svc.Logger) error {
	s.log = log
	log.Info("Service stopping")
	defer log.Info("Service stopped")
	if err := s.impl.Stop(); err != nil {
		return err
	}
	return s.ctl.Stop()
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode")
	svcFlag := flag.String("service", "", "Control the system service (actions: install, uninstall, start, stop)")
	flag.Parse()

	name := "NextDNSService"
	displayName := "NextDNS Service"
	desc := "NextDNS DNS53 to DoH proxy."

	var err error
	switch *svcFlag {
	case "install":
		err = svc.Install(name, displayName, desc)
	case "uninstall", "remove":
		err = svc.Remove(name)
	case "start":
		err = svc.Start(name)
	case "stop":
		err = svc.Stop(name)
	case "":
		err = run(*debug)
	default:
		fmt.Println("invalid service action")
	}
	if err != nil {
		fmt.Println(err)
	}
}

func run(debug bool) error {
	vers := updater.CurrentVersion()
	if vers == "" {
		vers = "dev"
	}

	up := &updater.Updater{
		URL: "https://storage.googleapis.com/nextdns_windows/info.json",
	}

	var s *nextdnsSvc
	broadcast := func(name string, data map[string]interface{}) {
		s.log.Info(fmt.Sprintf("send event: %v %v", name, data))
		if err := s.ctl.Broadcast(ctl.Event{Name: name, Data: data}); err != nil {
			s.log.Error(fmt.Sprintf("send event error: %v", err))
		}
	}
	s = &nextdnsSvc{
		ctl: ctl.Server{
			Namespace: "NextDNS",
			OnConnect: func(c net.Conn) {
				s.log.Info(fmt.Sprintf("UI Connect: %v", c))
			},
			OnDisconnect: func(c net.Conn) {
				s.log.Info(fmt.Sprintf("UI Disconnect: %v", c))
			},
			Handler: ctl.EventHandlerFunc(func(e ctl.Event) {
				s.log.Info(fmt.Sprintf("received event: %s %v", e.Name, e.Data))
				switch e.Name {
				case "open":
					// Use to open the GUI window in the existing instance of
					// the app when a duplicate instance is open.
					broadcast("open", nil)
				case "settings":
					if e.Data == nil {
						return
					}
					// Apply settings
					stg := settings.FromMap(e.Data)
					s.impl.SetConfigID(stg.Configuration)
					if stg.ReportDeviceName {
						s.impl.SetDeviceInfo(getHostname(), getModel(), getShortMachineID(), vers)
					} else {
						s.impl.SetDeviceInfo("", "", "", vers)
					}
					up.SetAutoRun(stg.CheckUpdates)

					// Switch connection status
					var err error
					if stg.Enabled {
						s.log.Info("Starting service")
						err = s.impl.Start()
					} else {
						s.log.Info("Stopping service")
						err = s.impl.Stop()
					}
					if err != nil {
						broadcast("status", map[string]interface{}{
							"state": s.impl.State(),
							"error": err.Error(),
						})
					}
				default:
					s.log.Error(fmt.Sprintf("invalid event: %v", e))
				}
			}),
		},
	}

	if windoh.Available() {
		s.impl = &windoh.Config{
			OnStateChange: func(state string) {
				broadcast("status", map[string]interface{}{"state": state})
			},
		}
	} else {
		s.impl = &proxy.Proxy{
			Upstream: "https://dns.nextdns.io/",
			// Bootstrap with a fake transport that avoid DNS lookup
			OnStateChange: func(state string) {
				broadcast("status", map[string]interface{}{"state": state})
			},
			// QueryLog: func(msgID uint16, qname string) {
			// 	s.log.Info(fmt.Sprintf("resolve %x %s", msgID, qname))
			// },
			InfoLog: func(msg string) {
				s.log.Info(msg)
			},
			ErrorLog: func(err error) {
				s.log.Error(fmt.Sprint(err))
			},
		}
	}

	s.ctl.ErrorLog = func(err error) {
		s.log.Error(fmt.Sprint(err))
	}
	up.OnUpgrade = func(newVersion string) {
		s.log.Info(fmt.Sprintf("upgrading from %s to %s", updater.CurrentVersion(), newVersion))
	}
	up.ErrorLog = func(err error) {
		s.log.Error(fmt.Sprint(err))
	}
	log.SetOutput(writerFunc(func(b []byte) (n int, err error) {
		s.log.Info(string(b))
		return len(b), nil
	}))

	return svc.Run(s, "NextDNSService", debug)
}

type writerFunc func(p []byte) (n int, err error)

func (w writerFunc) Write(p []byte) (n int, err error) {
	return w(p)
}

func getModel() string {
	cmd := exec.Command("wmic", "computersystem", "get", "model")
	b, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Remove Model\r\n prefix.
	for len(b) > 0 {
		if b[0] == '\n' {
			return strings.TrimSpace(string(b[1:]))
		}
		b = b[1:]
	}
	return ""
}

func getHostname() string {
	h, _ := os.Hostname()
	return h
}

func getShortMachineID() string {
	uuid, err := machineid.ID()
	if err != nil {
		return ""
	}
	sum := crc64.Checksum([]byte(uuid), crc64.MakeTable(crc64.ISO))
	return fmt.Sprintf("%x", sum)[:5]
}
