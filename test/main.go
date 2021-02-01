package main

import (
	"encoding/base64"
	"io/ioutil"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	b, _ := ioutil.ReadFile("/home/xy/Pictures/b.png")
	s := base64.RawStdEncoding.EncodeToString(b)
	println(s, len(b), len(s))
}
