package proxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nextdns/nextdns/resolver/endpoint"
	tun "github.com/nextdns/windows/tun"
)

const (
	StateStopped     = "stopped"
	StateStarting    = "starting"
	StateStarted     = "started"
	StateReasserting = "reasserting"
	StateStopping    = "stopping"
)

type Proxy struct {
	Upstream string

	ExtraHeaders http.Header

	OnStateChange func(state string)

	// Transport is the http.RoundTripper used to perform DoH requests.
	Transport http.RoundTripper

	// QueryLog specifies an optional log function called for each received query.
	QueryLog func(msgID uint16, qname string)

	// ErrorLog specifies an optional log function for errors. If not set,
	// errors are not reported.
	ErrorLog func(error)

	InfoLog func(string)

	mu    sync.Mutex
	tun   io.ReadWriteCloser
	state string
	stop  chan struct{}

	dedup dedup
}

func (p *Proxy) SetConfigID(id string) {
	p.Upstream = "https://dns.nextdns.io/" + id
}

func (p *Proxy) SetDeviceInfo(name, model, id, version string) {
	reportHdr := p.ExtraHeaders
	if reportHdr == nil {
		reportHdr = http.Header{}
		p.ExtraHeaders = reportHdr
	}
	if name != "" {
		reportHdr.Set("X-Device-Name", name)
	}
	if model != "" {
		reportHdr.Set("X-Device-Model", model)
	}
	if id != "" {
		reportHdr.Set("X-Device-Id", id)
	}
	reportHdr.Set("User-Agent", "nextdns-windows/"+version)
}

func (p *Proxy) State() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stateLocked()
}

func (p *Proxy) stateLocked() string {
	if p.state == "" {
		return StateStopped
	}
	return p.state
}

func (p *Proxy) setStateLocked(s string) {
	if p.state == s {
		return
	}
	p.state = s
	if p.OnStateChange != nil {
		p.OnStateChange(s)
	}
}

func (p *Proxy) Start() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stateLocked() != StateStopped {
		return nil // already started
	}
	p.setStateLocked(StateStarting)
	return p.startLocked()
}

func (p *Proxy) startLocked() (err error) {
	if p.tun, err = tun.OpenTunDevice("tun0", "192.0.2.43", "192.0.2.42", "255.255.255.0", []string{"192.0.2.42"}); err != nil {
		return err
	}
	p.Transport = p.nextdnsTransport()
	go p.run()
	return nil
}

// nextdnsTransport returns a endpoint.Manager configured to connect to NextDNS
// using different steering techniques.
func (p *Proxy) nextdnsTransport() http.RoundTripper {
	return &endpoint.Manager{
		Providers: []endpoint.Provider{
			// Prefer unicast routing.
			&endpoint.SourceURLProvider{
				SourceURL: "https://router.nextdns.io",
				Client: &http.Client{
					// Trick to avoid depending on DNS to contact the router API.
					Transport: endpoint.MustNew(fmt.Sprintf("https://router.nextdns.io#%s", []string{
						"216.239.32.21",
						"216.239.34.21",
						"216.239.36.21",
						"216.239.38.21",
					}[rand.Intn(3)])),
				},
			},
			// Fallback on anycast.
			endpoint.StaticProvider([]*endpoint.Endpoint{
				endpoint.MustNew("https://dns1.nextdns.io#45.90.28.0"),
				endpoint.MustNew("https://dns2.nextdns.io#45.90.30.0"),
			}),
			// Fallback on CDN fronting.
			endpoint.StaticProvider([]*endpoint.Endpoint{
				endpoint.MustNew("https://d1xovudkxbl47e.cloudfront.net"),
			}),
		},
		OnError: func(e *endpoint.Endpoint, err error) {
			if p.ErrorLog != nil {
				p.ErrorLog(fmt.Errorf("Endpoint failed: %s: %v", e.Hostname, err))
			}
		},
		OnChange: func(e *endpoint.Endpoint) {
			if p.InfoLog != nil {
				p.InfoLog(fmt.Sprintf("Switching endpoint: %s", e.Hostname))
			}
		},
	}
}

func (p *Proxy) Stop() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch p.stateLocked() {
	case StateStopped, StateStarting:
		return nil // already stopped
	}
	p.setStateLocked(StateStopping)
	if p.tun != nil {
		err = p.tun.Close()
		p.tun = nil
	}
	if p.stop != nil {
		close(p.stop)
		p.stop = nil
	}
	p.Transport = nil
	return err
}

func (p *Proxy) restartOrStop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch p.stateLocked() {
	case StateStopping:
		p.setStateLocked(StateStopped)
		return
	case StateStopped:
		// unexpected state
		return
	}
	p.setStateLocked(StateReasserting)
	for {
		time.Sleep(5 * time.Second)
		if err := p.startLocked(); err != nil {
			p.logErr(fmt.Errorf("restart err: %v", err))
			continue
		}
		break
	}
}

// doStart transitions to StateStarted. If the previous state wasn't
// StateStarting or StateReassessing, no transition happens and false is
// returned.
func (p *Proxy) doStart() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch p.stateLocked() {
	case StateStarting, StateReasserting:
		p.setStateLocked(StateStarted)
		return true
	default:
		return false
	}
}

func (p *Proxy) logQuery(msgID uint16, qname string) {
	if p.QueryLog != nil {
		p.QueryLog(msgID, qname)
	}
}

func (p *Proxy) logInfo(msg string) {
	if p.InfoLog != nil {
		p.InfoLog(msg)
	}
}
func (p *Proxy) logErr(err error) {
	if err != nil && p.ErrorLog != nil {
		p.ErrorLog(err)
	}
}

func (p *Proxy) run() {
	defer p.restartOrStop()

	// Setup firewall rules to avoid DNS leaking.
	// The process block forever and removes rules when killed.
	// We thus kill it as soon as we stop the proxy.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.unleak(ctx); err != nil {
		p.logErr(fmt.Errorf("cannot start dnsunleak: %v", err))
	}

	// Start the loop handling UDP packets received on the tun interface.
	const maxSize = 1500
	bpool := sync.Pool{
		New: func() interface{} {
			b := make([]byte, maxSize)
			return &b
		},
	}
	p.stop = make(chan struct{})
	// Isolate the reads in a goroutine so we can decide to bail when p.stop is
	// closed, even if tun.Read keeps blocking. This is to make sure we stop
	// dnsunleak and not leave the user with no DNS. This certainly hides a bug
	// in the tun library.
	packetIn := make(chan []byte)
	packetOut := make(chan []byte)
	tun := p.tun
	defer tun.Close()
	go func() {
		defer close(packetIn)
		if !p.doStart() {
			// Stop the start process
			return
		}
		for {
			buf := *bpool.Get().(*[]byte)
			n, err := tun.Read(buf[:maxSize]) // make sure we resize it to its max size
			if err != nil {
				if err != io.EOF {
					p.logErr(fmt.Errorf("tun read err: %v", err))
				}
				return
			}
			packetIn <- buf[:n]
		}
	}()
	go func() {
		for {
			var buf []byte
			var more bool
			select {
			case buf, more = <-packetOut:
				if !more {
					return
				}
			case <-p.stop:
				return
			}
			if _, err := tun.Write(buf); err != nil {
				p.logErr(fmt.Errorf("tun write error: %v", err))
				return
			}
			bpool.Put(&buf)
		}
	}()

	dnsIP := []byte{192, 0, 2, 42}
	for {
		var buf []byte
		var more bool
		select {
		case buf, more = <-packetIn:
		case <-p.stop:
			return
		}
		if !more {
			break
		}
		qsize := len(buf)
		if qsize <= 20 {
			bpool.Put(&buf)
			continue
		}
		if buf[9] != 17 {
			// Not UDP
			bpool.Put(&buf)
			continue
		}
		if !bytes.Equal(buf[16:20], dnsIP) {
			// Skip packet not directed to us.
			bpool.Put(&buf)
			continue
		}
		msgID := lazyMsgID(buf)
		if p.dedup.IsDup(msgID) {
			bpool.Put(&buf)
			// Skip duplicated query.
			continue
		}
		go func() {
			qname := lazyQName(buf)
			p.logQuery(msgID, qname)
			res, err := p.resolve(buf)
			if err != nil {
				p.logErr(fmt.Errorf("resolve: %x %v", msgID, err))
				return
			}
			buf = buf[:maxSize] // reset buf size to it's underlaying size
			rsize, err := readDNSResponse(res, buf)
			if err != nil {
				p.logErr(fmt.Errorf("readDNSResponse: %v", err))
				return
			}
			select {
			case packetOut <- buf[:rsize]:
			case <-p.stop:
			}
		}()
	}
}

func (p *Proxy) unleak(ctx context.Context) error {
	// Setup firewall rules to avoid DNS leaking.
	// The process block forever and removes rules when killed.
	// We thus kill it as soon as we stop the proxy.
	ex, _ := os.Executable()
	dnsunleakPath := filepath.Join(filepath.Dir(ex), "dnsunleak.exe")
	cmd := exec.CommandContext(ctx, dnsunleakPath)
	stdout, stdoutW := io.Pipe()
	stdinR, stdin := io.Pipe()
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stdoutW
	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			l := s.Text()
			p.logInfo(fmt.Sprintf("dnsunleak: %s", l))
		}
	}()
	go func() {
		<-ctx.Done()
		if proc := cmd.Process; proc != nil {
			p.logInfo("Killing dnsunleak")
			_, _ = stdin.Write([]byte{'\n'})
			_ = proc.Kill()
		}
	}()
	return cmd.Start()
}

func (p *Proxy) resolve(buf []byte) (io.ReadCloser, error) {
	req, err := http.NewRequest("POST", p.Upstream, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-packet")
	for name, hdrs := range p.ExtraHeaders {
		req.Header[name] = hdrs
	}
	rt := p.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	res, err := rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error code: %d", res.StatusCode)
	}
	return res.Body, nil
}

func readDNSResponse(r io.Reader, buf []byte) (int, error) {
	var n int
	for {
		nn, err := r.Read(buf[n:])
		n += nn
		if err != nil {
			if err == io.EOF {
				break
			}
			return -1, err
		}
		if n >= len(buf) {
			buf[2] |= 0x2 // mark response as truncated
			break
		}
	}
	return n, nil
}

// lazyMsgID parses the message ID from a DNS query wything trying to parse or
// validate the whole query.
func lazyMsgID(buf []byte) uint16 {
	if len(buf) < 30 {
		return 0
	}
	return uint16(buf[28])<<8 | uint16(buf[29])
}

// lazyQName parses the qname from a DNS query without trying to parse or
// validate the whole query.
func lazyQName(buf []byte) string {
	qn := &strings.Builder{}
	for n := 40; n <= len(buf) && buf[n] != 0; {
		end := n + 1 + int(buf[n])
		if end > len(buf) {
			// invalid qname, stop parsing
			break
		}
		qn.Write(buf[n+1 : end])
		qn.WriteByte('.')
		n = end
	}
	return qn.String()
}
