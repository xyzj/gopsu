package main

import (
	"fmt"

	"github.com/xyzj/gopsu"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	a := float64(331907000000)
	b := gopsu.Float642BcdBytesBigOrder(a, "%12.0f")
	println(gopsu.Bytes2String(b, "-"))
	c := gopsu.BcdBytes2Float64BigOrder(b, 0, true)
	println(fmt.Sprintf("%12.0f,%12.0f", a, c))
}
