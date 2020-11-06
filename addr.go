package gopsu

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// IPUint2String change ip int64 data to string format
func IPUint2String(ipnr uint) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ipnr>>24)&0xFF, (ipnr>>16)&0xFF, (ipnr>>8)&0xFF, ipnr&0xFF)
}

// IPInt642String change ip int64 data to string format
func IPInt642String(ipnr int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ipnr)&0xFF, (ipnr>>8)&0xFF, (ipnr>>16)&0xFF, (ipnr>>24)&0xFF)
}

// IPInt642Bytes change ip int64 data to string format
func IPInt642Bytes(ipnr int64) []byte {
	return []byte{byte((ipnr) & 0xFF), byte((ipnr >> 8) & 0xFF), byte((ipnr >> 16) & 0xFF), byte((ipnr >> 24) & 0xFF)}
}

// IPUint2Bytes change ip int64 data to string format
func IPUint2Bytes(ipnr int64) []byte {
	return []byte{byte((ipnr >> 24) & 0xFF), byte((ipnr >> 16) & 0xFF), byte((ipnr >> 8) & 0xFF), byte((ipnr) & 0xFF)}
}

// IP2Uint change ip string data to int64 format
func IP2Uint(ipnr string) uint {
	// ex := errors.New("wrong ip address format")
	bits := strings.Split(ipnr, ".")
	if len(bits) != 4 {
		return 0
	}
	var intip uint
	for k, v := range bits {
		i, ex := strconv.Atoi(v)
		if ex != nil || i > 255 || i < 0 {
			return 0
		}
		intip += uint(i) << uint(8*(3-k))
	}
	return intip
}

// IP2Int64 change ip string data to int64 format
func IP2Int64(ipnr string) int64 {
	// ex := errors.New("wrong ip address format"
	bits := strings.Split(ipnr, ".")
	if len(bits) != 4 {
		return 0
	}
	var intip uint
	for k, v := range bits {
		i, ex := strconv.Atoi(v)
		if ex != nil || i > 255 || i < 0 {
			return 0
		}
		intip += uint(i) << uint(8*(k))
	}
	return int64(intip)
}

// RealIP 返回本机的v4或v6ip
func RealIP(v6first bool) string {
	if v6first {
		if ip := ExternalIPV6(); ip != "" {
			return ip
		}
	}
	return ExternalIP()
}

// ExternalIP 返回v4地址
func ExternalIP() string {
	v4, v6, err := GlobalIPs()
	if err != nil {
		return ""
	}
	if len(v4) > 0 {
		return v4[0]
	}
	if len(v6) > 0 {
		return v6[0]
	}
	return ""
}

// ExternalIPV6 返回v6地址
func ExternalIPV6() string {
	_, v6, err := GlobalIPs()
	if err != nil {
		return ""
	}
	if len(v6) > 0 {
		return v6[0]
	}
	return ""
}

// GlobalIPs 返回所有可访问ip
// ipv4 list,ipv6 list
func GlobalIPs() ([]string, []string, error) {
	var v4, v6 = make([]string, 0), make([]string, 0)
	s, err := net.InterfaceAddrs()
	if err != nil {
		return v4, v6, err
	}
	for _, a := range s {
		var ip net.IP
		switch addr := a.(type) {
		case *net.IPAddr:
			ip = addr.IP
		case *net.IPNet:
			ip = addr.IP
		default:
			continue
		}
		if ip.To4().String() != "<nil>" {
			if ip.IsGlobalUnicast() {
				v4 = append(v4, ip.To4().String())
			}
			continue
		}
		if ip.To16().String() != "<nil>" {
			if ip.IsGlobalUnicast() {
				v6 = append(v6, "["+ip.To16().String()+"]")
			}
		}
	}
	return v4, v6, nil
}
