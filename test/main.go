package main

import (
	"github.com/xyzj/gopsu"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	println(gopsu.GetRandomString(16))
	println(gopsu.GetRandomString(16))
}
