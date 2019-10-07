package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/ProtonMail/go-autostart"
	"github.com/getlantern/systray"
	"github.com/nextdns/nextdns/proxy"
	"golang.org/x/net/http2"

	"github.com/rs/nextdns-windows/dns"
	"github.com/rs/nextdns-windows/icon"
	"github.com/rs/nextdns-windows/settings"
)

func displayError(err error) {
	if err == nil {
		return
	}
	fmt.Println(err)
	// TODO: show a window
}

func main() {
	s := settings.Load()

	const endpointBase = "https://45.90.28.0/"
	endpoint := endpointBase + s.Configuration

	self, err := os.Executable()
	if err != nil {
		displayError(err)
	} else {
		println(self)
		app := &autostart.App{
			Name:        "NextDNS",
			DisplayName: "NextDNS",
			Exec:        []string{self},
		}
		if !app.IsEnabled() {
			if err := app.Enable(); err != nil {
				displayError(err)
			}
		}
	}

	go func() {
		p := proxy.Proxy{
			Addr: "127.0.0.1:53",
			Upstream: func(qname string) string {
				return endpoint
			},
			Client: &http.Client{
				Transport: &http2.Transport{
					TLSClientConfig: &tls.Config{
						ServerName: "dns.nextdns.io",
					},
				},
			},
		}
		if err := p.ListenAndServe(context.Background()); err != nil {
			displayError(err)
		}
	}()

	systray.Run(func() {
		systray.SetIcon(icon.Data)
		systray.SetTitle("NextDNS")
		systray.SetTooltip("NextDNS Client")

		connect := systray.AddMenuItem("Connect", "Toggle NextDNS")
		var connected bool
		updateLabel := func() {
			var err error
			connected, err = dns.Enabled()
			if err != nil {
				displayError(err)
			}
			if connected {
				connect.SetTitle("Disconnect")
			} else {
				connect.SetTitle("Connect")
			}
		}
		updateLabel()
		go func() {
			for {
				<-connect.ClickedCh
				if !connected {
					displayError(dns.Enable())
				} else {
					displayError(dns.Disable())
				}
				updateLabel()
			}
		}()

		preferences := systray.AddMenuItem("Preferences...", "NextDNS preferences")
		go func() {
			for {
				<-preferences.ClickedCh
				displayError(s.Edit())
				endpoint = endpointBase + s.Configuration
			}
		}()

		quit := systray.AddMenuItem("Quit", "Quit NextDNS")
		go func() {
			<-quit.ClickedCh
			systray.Quit()
		}()
	}, func() {})
}
