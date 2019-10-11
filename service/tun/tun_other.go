//+build !windows

package tun

import (
	"errors"
	"io"
)

func OpenTunDevice(name, addr, gw, mask string, dns []string) (io.ReadWriteCloser, error) {
	return nil, errors.New("not implemented")
}
