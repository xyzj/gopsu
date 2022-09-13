package main

import (
	"fmt"

	"github.com/xyzj/go-pinyin"
)

func aaa(a, b, c string, d, e int) {
	println(fmt.Sprintf("%s, %s, %s, --- %d %d", a, b, c, d, e))
}

func main() {
	s := "长江12南ABc路"
	println(s)
	println(pinyin.XPinyin(s, pinyin.ReturnNormal))
}
