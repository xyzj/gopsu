package main

import (
	"net"

	"github.com/xyzj/gopsu"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	println(gopsu.RealIP(true))
	println(gopsu.RealIP(false))
	a, b, err := net.SplitHostPort("[240e:e5:8001:1856:4d6a:8a0c:7814:a622]:")
	if err != nil {
		println(err.Error())
	}
	println(a, "=="+b+"--")
}
