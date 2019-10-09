package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"github.com/getlantern/systray"
	"golang.org/x/net/http2"

	"github.com/rs/nextdns-windows/icon"
	"github.com/rs/nextdns-windows/proxy"
	"github.com/rs/nextdns-windows/settings"
)

func displayError(err error) {
	if err == nil {
		return
	}
	log.Print(err)
	// TODO: show a window
}

func main() {
	log.SetOutput(os.Stdout)
	s := settings.Load()
	p := &proxy.Proxy{
		Client: &http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: &tls.Config{
					ServerName: "dns.nextdns.io",
				},
			},
		},
	}
	const upstreamBase = "https://45.90.28.0/"
	p.Upstream = upstreamBase + s.Configuration

	systray.Run(func() {
		systray.SetIcon(icon.Data)
		systray.SetTitle("NextDNS")
		systray.SetTooltip("NextDNS Client")

		connect := systray.AddMenuItem("Enable", "Toggle NextDNS")
		var started bool
		updateLabel := func() {
			var err error
			started, err = p.Started()
			if err != nil {
				displayError(err)
			}
			if started {
				connect.SetTitle("Disable")
			} else {
				connect.SetTitle("Enable")
			}
		}
		p.OnStateChange = updateLabel
		updateLabel()
		go func() {
			for {
				<-connect.ClickedCh
				if !started {
					displayError(p.Start())
				} else {
					displayError(p.Stop())
				}
				updateLabel()
			}
		}()

		preferences := systray.AddMenuItem("Preferences...", "NextDNS preferences")
		go func() {
			for {
				<-preferences.ClickedCh
				displayError(s.Edit())
				p.Upstream = upstreamBase + s.Configuration
			}
		}()

		quit := systray.AddMenuItem("Quit", "Quit NextDNS")
		go func() {
			<-quit.ClickedCh
			systray.Quit()
		}()
	}, func() {})
}
