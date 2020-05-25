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
	s := "gopsu.GetRandomString(16)"
	c := gopsu.GetNewCryptoWorker(gopsu.CryptoAES128CBC)
	c.SetKey(gopsu.GetRandomString(16), gopsu.GetRandomString(16))
	println(s)
	s = c.EncryptNoTail(s)
	println(s)
}
