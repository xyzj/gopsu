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
	var aaa = gopsu.NewCache(10)
	for i := 0; i < 111; i++ {
		println(i, aaa.SetWithHold(i, "v interface{}", 59000, 60000))
	}
	for {
		time.Sleep(time.Second)
		println(aaa.Len())
	}
}
