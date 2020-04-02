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
	cw := gopsu.GetNewCryptoWorker(gopsu.CryptoHMACSHA1)
	cw.SetSignKey([]byte("key"))
	println(cw.Hash([]byte("qwerty")))
	println(cw.Hash([]byte("qwerty")))
	println(cw.Hash([]byte("qwerty")))
	println(cw.Hash([]byte("qwerty")))
}
