package main

import (
	"context"
	"flag"
	"fmt"
	"hash/crc64"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/nextdns/nextdns/endpoint"

	"github.com/rs/nextdns-windows/ctl"
	"github.com/rs/nextdns-windows/proxy"
	"github.com/rs/nextdns-windows/settings"
	"github.com/rs/nextdns-windows/svc"
	"github.com/rs/nextdns-windows/updater"
)

const upstreamBase = "https://dns.nextdns.io/"

type proxySvc struct {
	proxy.Proxy
	router endpoint.Manager
	ctl    ctl.Server
	log    svc.Logger
}

func (p *proxySvc) Start(log svc.Logger) error {
	p.log = log
	log.Info("Service starting")
	defer log.Info("Service started")
	return p.ctl.Start()
}

func (p *proxySvc) Stop(log svc.Logger) error {
	p.log = log
	log.Info("Service stopping")
	defer log.Info("Service stopped")
	if err := p.Proxy.Stop(); err != nil {
		return err
	}
	return p.ctl.Stop()
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
	hdrs := http.Header{}
	vers := updater.CurrentVersion()
	if vers == "" {
		vers = "dev"
	}
	hdrs.Set("User-Agent", "nextdns-windows/"+vers)

	reportHdr := hdrs.Clone()
	if model := getModel(); model != "" {
		reportHdr.Set("X-Device-Model", model)
	}
	if hostname := getHostname(); hostname != "" {
		reportHdr.Set("X-Device-Name", hostname)
	}
	if hostID := getShortMachineID(); hostID != "" {
		reportHdr.Set("X-Device-Id", hostID)
	}

	up := &updater.Updater{
		URL: "https://storage.googleapis.com/nextdns_windows/info.json",
	}

	var p *proxySvc
	broadcast := func(name string, data map[string]interface{}) {
		p.log.Info(fmt.Sprintf("send event: %v %v", name, data))
		if err := p.ctl.Broadcast(ctl.Event{Name: name, Data: data}); err != nil {
			p.log.Error(fmt.Sprintf("send event error: %v", err))
		}
	}
	p = &proxySvc{
		Proxy: proxy.Proxy{
			Upstream:     upstreamBase,
			ExtraHeaders: hdrs,
			// Bootstrap with a fake transport that avoid DNS lookup
			Transport: endpoint.NewTransport(endpoint.New("dns.nextdns.io", "", "45.90.28.0")),
			OnStateChange: func(started bool) {
				broadcast("status", map[string]interface{}{"enabled": started})
			},
		},
		router: endpoint.Manager{
			Providers: []endpoint.Provider{
				// Prefer unicast routing.
				endpoint.SourceURLProvider{
					SourceURL: "https://router.nextdns.io",
					Client: &http.Client{
						// Trick to avoid depending on DNS to contact the router API.
						Transport: endpoint.NewTransport(
							endpoint.New("router.nextdns.io", "", []string{
								"216.239.32.21",
								"216.239.34.21",
								"216.239.36.21",
								"216.239.38.21",
							}[rand.Intn(3)])),
					},
				},
				// Fallback on anycast.
				endpoint.StaticProvider(endpoint.New("dns1.nextdns.io", "", "45.90.28.0")),
				endpoint.StaticProvider(endpoint.New("dns2.nextdns.io", "", "45.90.30.0")),
				// Fallback on CDN fronting.
				endpoint.StaticProvider(endpoint.New("d1xovudkxbl47e.cloudfront.net", "", "")),
			},
			OnError: func(e endpoint.Endpoint, err error) {
				p.log.Warn(fmt.Sprintf("Endpoint failed: %s: %v", e.Hostname, err))
			},
			OnChange: func(e endpoint.Endpoint, rt http.RoundTripper) {
				p.log.Info(fmt.Sprintf("Switching endpoint: %s", e.Hostname))
				p.Transport = rt
			},
		},
		ctl: ctl.Server{
			Namespace: "NextDNS",
			Handler: ctl.EventHandlerFunc(func(e ctl.Event) {
				p.log.Info(fmt.Sprintf("received event: %s %v", e.Name, e.Data))
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
					s := settings.FromMap(e.Data)
					p.Upstream = upstreamBase + s.Configuration
					if s.ReportDeviceName {
						p.ExtraHeaders = reportHdr
					} else {
						p.ExtraHeaders = hdrs
					}
					up.SetAutoRun(s.CheckUpdates)

					// Switch connection status
					var err error
					if s.Enabled {
						p.log.Info("Starting proxy")
						err = p.Proxy.Start()
						if err == nil {
							go func() {
								ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
								defer cancel()
								if err := p.router.Test(ctx); err != nil {
									p.ErrorLog(fmt.Errorf("router: %v", err))
								}
							}()
						}
					} else {
						p.log.Info("Stopping proxy")
						err = p.Proxy.Stop()
					}
					if err != nil {
						if err == proxy.ErrAlreadyStarted || err == proxy.ErrAlreadyStopped {
							// Make sure the UI gets a status update if we are
							// already in the requested state.
							broadcast("status", map[string]interface{}{"enabled": p.Proxy.Started()})
						} else {
							p.log.Error(fmt.Sprintf("proxy: %v", err))
							broadcast("error", map[string]interface{}{"error": err.Error()})
						}
					}
				default:
					p.ErrorLog(fmt.Errorf("invalid event: %v", e))
				}
			}),
		},
	}

	// p.QueryLog = func(msgID uint16, qname string) {
	// 	p.log.Info(fmt.Sprintf("resolve %x %s", msgID, qname))
	// }
	p.InfoLog = func(msg string) {
		p.log.Info(msg)
	}
	p.ErrorLog = func(err error) {
		p.log.Error(fmt.Sprint(err))
	}
	p.ctl.ErrorLog = func(err error) {
		p.log.Error(fmt.Sprint(err))
	}
	up.OnUpgrade = func(newVersion string) {
		p.log.Info(fmt.Sprintf("upgrading from %s to %s", updater.CurrentVersion(), newVersion))
	}
	up.ErrorLog = func(err error) {
		p.log.Error(fmt.Sprint(err))
	}

	return svc.Run(p, "NextDNSService", debug)
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
