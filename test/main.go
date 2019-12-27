package main

import (
	"fmt"

	"github.com/xyzj/gopsu"
)

// 启动文件 main.go
func main() {

	println(fmt.Sprintf("%.1f", gopsu.BcdBytes2Bin([]byte{0x01, 0xa8}, 0, false)))
}
