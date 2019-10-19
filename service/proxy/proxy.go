package proxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	tun "github.com/rs/nextdns-windows/tun"
)

type Proxy struct {
	Upstream string
	Hostname string
	HostID   string

	OnStateChange func()

	// Transport is the http.RoundTripper used to perform DoH requests.
	Transport http.RoundTripper

	// QueryLog specifies an optional log function called for each received query.
	QueryLog func(qname string)

	// ErrorLog specifies an optional log function for errors. If not set,
	// errors are not reported.
	ErrorLog func(error)

	InfoLog func(string)

	mu  sync.Mutex
	tun io.ReadWriteCloser
}

func (p *Proxy) Started() (bool, error) {
	return p.tun != nil, nil
}

func (p *Proxy) Start() (err error) {
	if p.tun, err = tun.OpenTunDevice("tun0", "192.0.2.43", "192.0.2.42", "255.255.255.0", []string{"192.0.2.42"}); err != nil {
		return err
	}
	go p.run()
	return nil
}

func (p *Proxy) Stop() (err error) {
	if p.tun != nil {
		return p.tun.Close()
	}
	return nil
}

func (p *Proxy) logQuery(qname string) {
	if p.QueryLog != nil {
		p.QueryLog(qname)
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
	if p.OnStateChange != nil {
		p.OnStateChange()
	}
	defer func() {
		p.tun = nil
		if p.OnStateChange != nil {
			p.OnStateChange()
		}
	}()

	// Setup firewall rules to avoid DNS leaking.
	// The process block forever and removes rules when killed.
	// We thus kill it as soon as we stop the proxy.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.unleak(ctx); err != nil {
		p.logErr(fmt.Errorf("cannot start dnsunleak: %v", err))
	}

	// Start the loop handling UDP packets received on the tun interface.
	bpool := sync.Pool{
		New: func() interface{} {
			b := make([]byte, 1500)
			return &b
		},
	}
	dnsIP := []byte{192, 0, 2, 42}
	for {
		buf := *bpool.Get().(*[]byte)
		qsize, err := p.tun.Read(buf)
		if err != nil {
			p.logErr(fmt.Errorf("tun read err: %v", err))
			break
		}
		if qsize <= 20 {
			bpool.Put(&buf)
			continue
		}
		if buf[9] != 17 {
			// Not UDP
			continue
		}
		if !bytes.Equal(buf[16:20], dnsIP) {
			// Skip packet not directed to us.
			continue
		}
		go func() {
			defer bpool.Put(&buf)
			qname := lazyQName(buf)
			p.logQuery(qname)
			res, err := p.resolve(buf[:qsize])
			if err != nil {
				p.logErr(fmt.Errorf("resolve: %v", err))
				return
			}
			rsize, err := readDNSResponse(res, buf)
			if err != nil {
				p.logErr(fmt.Errorf("readDNSResponse: %v", err))
				return
			}
			p.mu.Lock()
			defer p.mu.Unlock()
			if _, err := p.tun.Write(buf[:rsize]); err != nil {
				p.logErr(fmt.Errorf("tun write error: %v", err))
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
	r, w := io.Pipe()
	cmd.Stdin = os.Stdin
	cmd.Stdout = w
	cmd.Stderr = w
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			l := s.Text()
			p.logInfo(fmt.Sprintf("dnsunleak: %s", l))
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
	if p.Hostname != "" {
		req.Header.Set("X-Device-Name", p.Hostname)
	}
	if p.HostID != "" {
		req.Header.Set("X-Device-Id", p.HostID)
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
