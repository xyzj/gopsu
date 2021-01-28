package main

import (
	"encoding/base64"
	"strings"

	"github.com/xyzj/gopsu"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	s := strings.Split("thunder://QUFodHRwOi8vZG93bjIub2tkb3duMTAuY29tLzIwMjEwMTI3LzMyNDBfZjJiNjg4MmMvw/fWzr+qu68g0MLKrsDJ1ezMvcz7RVAwNi5tcDRaWg==", ":")
	ss, _ := base64.StdEncoding.DecodeString(s[1][2:])
	a, _ := gopsu.GbkToUtf8(ss[2 : len(ss)-2])
	println(string(ss), "\n", string(a))
}
