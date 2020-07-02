package main

import (
	"time"

	"github.com/xyzj/gopsu"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	var aaa = gopsu.NewCache(100)
	for i := 0; i < 111; i++ {
		println(i, aaa.Set(i, "v interface{}", 6000))
	}
	for {
		time.Sleep(time.Second)
		println(aaa.Len())
	}
}
