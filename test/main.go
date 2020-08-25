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
	log := gopsu.NewLogger("lib", "log.txt", 20, 10)
	for {
		log.Info(time.Now().Format(gopsu.LongTimeFormat))
		time.Sleep(time.Second * 5)
	}
}
