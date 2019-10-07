package dns

import (
	"fmt"
	"io/ioutil"
	"strings"
	"syscall"
	"unsafe"

	"github.com/StackExchange/wmi"
)

func Enabled() (bool, error) {
	ifaces, err := getAllInterfaces()
	if err != nil {
		return false, err
	}
	for _, iface := range ifaces {
		if hasIP(iface.DNSServerSearchOrder, "127.0.0.1") {
			return true, nil
		}
	}
	return false, nil
}

func Enable() error {
	ifaces, err := getAllInterfaces()
	if err != nil {
		return err
	}
	var cmds []string
	for _, iface := range ifaces {
		if !hasIP(iface.DNSServerSearchOrder, "127.0.0.1") {
			cmds = append(cmds, fmt.Sprintf("interface ipv4 add dnsservers \"%s\" 127.0.0.1 index=0 validate=no", iface.Name))
		}
		if hasIPv6(iface.IPAddress) && !hasIP(iface.DNSServerSearchOrder, "::1") {
			cmds = append(cmds, fmt.Sprintf("interface ipv6 add dnsservers \"%s\" ::1 index=0 validate=no", iface.Name))
		}
	}
	return netsh(cmds)
}

func Disable() error {
	ifaces, err := getAllInterfaces()
	if err != nil {
		return err
	}
	var cmds []string
	for _, iface := range ifaces {
		cmds = append(cmds, fmt.Sprintf("interface ipv4 delete dnsservers \"%s\" 127.0.0.1", iface.Name))
		if hasIPv6(iface.IPAddress) {
			cmds = append(cmds, fmt.Sprintf("interface ipv6 delete dnsservers \"%s\" ::1", iface.Name))
		}
	}
	return netsh(cmds)
}

type win32NetworkAdapterConfiguration struct {
	Index                uint32
	IPAddress            []string
	DNSServerSearchOrder []string
}

type win32NetworkAdapter struct {
	Index           uint32
	NetConnectionID string
}

type Interface struct {
	Name                 string
	IPAddress            []string
	DNSServerSearchOrder []string
}

func getAllInterfaces() ([]Interface, error) {
	var netIfConf []win32NetworkAdapterConfiguration
	if err := wmi.Query("SELECT * FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled = True", &netIfConf); err != nil {
		return nil, err
	}
	var netIf []win32NetworkAdapter
	if err := wmi.Query(fmt.Sprintf("SELECT * FROM Win32_NetworkAdapter"), &netIf); err != nil {
		return nil, err
	}
	var ifaces []Interface
	for _, ifConf := range netIfConf {
		for _, ifInfo := range netIf {
			if ifConf.Index == ifInfo.Index {
				ifaces = append(ifaces, Interface{
					Name:                 ifInfo.NetConnectionID,
					IPAddress:            ifConf.IPAddress,
					DNSServerSearchOrder: ifConf.DNSServerSearchOrder,
				})
				break
			}
		}
	}
	return ifaces, nil
}

func hasIP(ips []string, ip string) bool {
	for _, _ip := range ips {
		if _ip == ip {
			return true
		}
	}
	return false
}

func hasIPv6(ips []string) bool {
	return false
	for _, ip := range ips {
		if strings.IndexByte(ip, ':') != -1 {
			return true
		}
	}
	return false
}

func netsh(cmds []string) error {
	f, err := ioutil.TempFile("", "nextdns-")
	if err != nil {
		return err
	}
	for _, cmd := range cmds {
		f.Write(append([]byte(cmd), '\n'))
	}
	f.Close()
	// defer os.Remove(f.Name())
	var hand uintptr = uintptr(0)
	var operator uintptr = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("runas")))
	var fpath uintptr = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("netsh.exe")))
	var param uintptr = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("-f " + f.Name())))
	var dirpath uintptr = uintptr(0)
	var ncmd uintptr = uintptr(0)
	shell32 := syscall.NewLazyDLL("shell32.dll")
	ShellExecuteW := shell32.NewProc("ShellExecuteW")
	_, _, err = ShellExecuteW.Call(hand, operator, fpath, param, dirpath, ncmd)
	return err
}
