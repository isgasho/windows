package windoh

import (
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	StateStopped = "stopped"
	StateStarted = "started"
)

func Available() bool {
	_, err := netsh("dns", "show", "encryption")
	return err == nil
}

type Config struct {
	id          string
	deviceName  string
	deviceModel string
	deviceID    string

	mu    sync.Mutex
	state string

	OnStateChange func(state string)
}

func (c *Config) State() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == "" {
		return StateStopped
	}
	return c.state
}

func (c *Config) setState(s string) {
	if c.state == s {
		return
	}
	c.state = s
	if c.OnStateChange != nil {
		c.OnStateChange(s)
	}
}

func (c *Config) SetConfigID(id string) {
	c.id = id
}

func (c *Config) SetDeviceInfo(name, model, id, version string) {
	c.deviceName = name
	c.deviceModel = model
	c.deviceID = id
}

func (c *Config) Start() error {
	ids, err := interfaces()
	if err != nil {
		return err
	}
	url := c.url()
	first := true
	for _, ip := range []string{"45.90.28.0", "45.90.30.0"} {
		if _, err := netsh("dns", "set", "encryption",
			"server="+ip,
			"dohtemplate="+url,
			"autoupgrade=yes",
			"udpfallback=no"); err != nil {
			return fmt.Errorf("set DoH template for %s: %w", ip, err)
		}
		if first {
			first = false
			for _, id := range ids {
				if _, err := netsh("dns", "set", "dnsserver",
					fmt.Sprintf("name=%d", id),
					"source=static",
					"address="+ip,
					"register=both"); err != nil {
					return fmt.Errorf("set DNS server for interface %d to %s: %w", id, ip, err)
				}
			}
		} else {
			for _, id := range ids {
				if _, err := netsh("dns", "add", "dnsserver", fmt.Sprintf("name=%d", id), "address="+ip); err != nil {
					return fmt.Errorf("add DNS server for interface %d to %s: %w", id, ip, err)
				}
			}
		}
	}
	if _, err := netsh("dns", "set", "global", "doh=yes"); err != nil {
		return fmt.Errorf("set global DoH: %w", err)
	}
	c.setState(StateStarted)
	return nil
}

func (c *Config) Stop() error {
	ids, err := interfaces()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := netsh("dns", "set", "dnsserver",
			fmt.Sprintf("name=%d", id),
			"source=dhcp"); err != nil {
			return err
		}
	}
	c.setState(StateStopped)
	return nil
}

func (c *Config) url() string {
	if c.deviceName == "" {
		return fmt.Sprintf("https://windows.dns.nextdns.io/%s", c.id)
	} else {
		return fmt.Sprintf("https://windows.dns.nextdns.io/%s/%s/%s/%s",
			c.id,
			url.PathEscape(c.deviceName),
			url.PathEscape(c.deviceModel),
			url.PathEscape(c.deviceID),
		)
	}
}

func interfaces() ([]int, error) {
	out, err := netsh("interface", "ipv4", "show", "interfaces")
	if err != nil {
		return nil, err
	}
	var ids []int
	for _, line := range strings.Split(string(out), "\n") {
		flds := strings.Fields(line)
		if len(flds) < 1 {
			continue
		}
		if i, err := strconv.ParseInt(flds[0], 10, 32); err == nil {
			ids = append(ids, int(i))
		}
	}
	return ids, nil
}

func netsh(args ...string) ([]byte, error) {
	out, err := exec.Command("netsh.exe", args...).Output()
	if err != nil {
		err = fmt.Errorf("%s: %w", string(out), err)
	}
	return out, err
}
