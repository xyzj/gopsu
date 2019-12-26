package main

import (
	"fmt"

	"github.com/xyzj/gopsu"
)

// 启动文件 main.go
func main() {
	a, b, c := gopsu.GPS2DFM(116.83)
	println(a, b, fmt.Sprintf("%.02f", c))
	println(fmt.Sprintf("%.02f", gopsu.DFM2GPS(116, 49, 48.00)))
}
