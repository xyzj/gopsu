package main

import (
	"fmt"
	"net"

	"github.com/xyzj/gopsu/pinyin"
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
	s := "12er应用设备 / 智慧交通 / 停车位监测设备"
	println(pinyin.XPinyinMatch(s, "zhjt"))
}
