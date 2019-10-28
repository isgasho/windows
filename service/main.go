package main

import (
	"context"
	"flag"
	"fmt"
	"hash/crc64"
	stdlog "log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/kardianos/service"
	"github.com/nextdns/nextdns/endpoint"

	"github.com/rs/nextdns-windows/ctl"
	"github.com/rs/nextdns-windows/proxy"
	"github.com/rs/nextdns-windows/settings"
	"github.com/rs/nextdns-windows/updater"
)

const upstreamBase = "https://dns.nextdns.io/"

var log service.Logger

type proxySvc struct {
	proxy.Proxy
	router endpoint.Manager
	ctl    ctl.Server
}

func (p *proxySvc) Start(s service.Service) error {
	log.Info("Service starting")
	defer log.Info("Service started")
	return p.ctl.Start()
}

func (p *proxySvc) Stop(s service.Service) error {
	log.Info("Service stopping")
	defer log.Info("Service stopped")
	if err := p.Proxy.Stop(); err != nil {
		return err
	}
	return p.ctl.Stop()
}

func main() {
	stdlog.SetOutput(os.Stdout)
	svcFlag := flag.String("service", "", fmt.Sprintf("Control the system service.\nValid actions: %s", strings.Join(service.ControlAction[:], ", ")))
	flag.Parse()

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
		log.Infof("send event: %v %v", name, data)
		if err := p.ctl.Broadcast(ctl.Event{Name: name, Data: data}); err != nil {
			_ = log.Errorf("send event error: %v", err)
		}
	}
	p = &proxySvc{
		proxy.Proxy{
			Upstream:     upstreamBase,
			ExtraHeaders: hdrs,
			// Bootstrap with a fake transport that avoid DNS lookup
			Transport: endpoint.NewTransport(endpoint.New("dns.nextdns.io", "", "45.90.28.0")),
			OnStateChange: func(started bool) {
				broadcast("status", map[string]interface{}{"enabled": started})
			},
		},
		endpoint.Manager{
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
				_ = log.Warningf("Endpoint failed: %s: %v", e.Hostname, err)
			},
			OnChange: func(e endpoint.Endpoint, rt http.RoundTripper) {
				_ = log.Infof("Switching endpoint: %s", e.Hostname)
				p.Transport = rt
			},
		},
		ctl.Server{
			Namespace: "NextDNS",
			Handler: ctl.EventHandlerFunc(func(e ctl.Event) {
				_ = log.Infof("received event: %s %v", e.Name, e.Data)
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
						if !p.Proxy.Started() {
							go func() {
								ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
								defer cancel()
								if err := p.router.Test(ctx); err != nil {
									p.ErrorLog(fmt.Errorf("router: %v", err))
								}
							}()
							log.Info("Starting proxy")
							err = p.Proxy.Start()
						} else {
							log.Info("Proxy already started")
						}
					} else {
						log.Info("Stopping proxy")
						err = p.Proxy.Stop()
					}
					if err != nil {
						_ = log.Errorf("proxy: %v", err)
						broadcast("error", map[string]interface{}{"error": err.Error()})
					}
				default:
					p.ErrorLog(fmt.Errorf("invalid event: %v", e))
				}
			}),
		},
	}

	svcConfig := &service.Config{
		Name:        "NextDNSService",
		DisplayName: "NextDNS Service",
		Description: "NextDNS DNS53 to DoH proxy.",
	}
	s, err := service.New(p, svcConfig)
	if err != nil {
		stdlog.Fatal(err)
	}
	errs := make(chan error, 5)
	if log, err = s.Logger(errs); err != nil {
		stdlog.Fatal(err)
	}
	go func() {
		for {
			err := <-errs
			if err != nil {
				stdlog.Print(err)
			}
		}
	}()
	p.QueryLog = func(msgID uint16, qname string) {
		_ = log.Infof("resolve %x %s", msgID, qname)
	}
	p.InfoLog = func(msg string) {
		_ = log.Info(msg)
	}
	p.ErrorLog = func(err error) {
		_ = log.Error(err)
	}
	p.ctl.ErrorLog = func(err error) {
		_ = log.Error(err)
	}
	up.OnUpgrade = func(newVersion string) {
		_ = log.Infof("upgrading from %s to %s", updater.CurrentVersion(), newVersion)
	}
	up.ErrorLog = func(err error) {
		_ = log.Error(err)
	}
	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			stdlog.Fatal(err)
		}
		return
	}
	if err = s.Run(); err != nil {
		_ = log.Error(err)
	}
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
