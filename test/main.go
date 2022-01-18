package main

import (
	"fmt"
	"net"

	"github.com/xyzj/gopsu"
)

func GetAvailablePort() (int, error) {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", "0.0.0.0"))
	if err != nil {
		return 0, err
	}
	var listener *net.TCPListener
	var found = false
	for i := 0; i < 100; i++ {
		listener, err = net.ListenTCP("tcp", address)
		if err != nil {
			continue
		}
		found = true
	}
	defer listener.Close()
	if !found {
		return 0, fmt.Errorf("could not find a useful port")
	}
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// 启动文件 main.go
func main() {
	s := []byte{0x56, 0x31, 0x2e, 0x30, 0x2e, 0x30, 0x30, 0x33, 0x00, 0x04, 0x00, 0x00, 0x00}
	println(string(s))
	a := gopsu.CalcCRC32(s, false)
	println(gopsu.Bytes2String(a, "-"))
}
