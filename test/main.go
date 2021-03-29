package main

import (
	"fmt"

	"github.com/xyzj/gopsu"
)

// 启动文件 main.go
func main() {
	str := "你好你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC你好 ABC ABC"
	a := gopsu.SplitStringWithLen(str, 67)
	println(len(a), fmt.Sprintf("%+v", a))
	println(gopsu.String2Int("", 0))
}
