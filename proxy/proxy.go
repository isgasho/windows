package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	tun "github.com/rs/nextdns-windows/tun"
)

type Proxy struct {
	Upstream      string
	Client        *http.Client
	OnStateChange func()

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
	return p.tun.Close()
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
			log.Printf("tun read err: %v", err)
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
			log.Print("resolve ", qname)
			res, err := p.resolve(buf[:qsize])
			if err != nil {
				log.Printf("resolve: %v", err)
				return
			}
			rsize, err := readDNSResponse(res, buf)
			if err != nil {
				log.Printf("readDNSResponse: %v", err)
				return
			}
			if _, err := p.tun.Write(buf[:rsize]); err != nil {
				log.Printf("tun write error: %v", err)
			}
		}()
	}
}

func (p *Proxy) resolve(buf []byte) (io.ReadCloser, error) {
	req, err := http.NewRequest("POST", p.Upstream, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-packet")
	c := p.Client
	if c == nil {
		c = http.DefaultClient
	}
	res, err := c.Do(req)
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
