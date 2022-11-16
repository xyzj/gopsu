package main

import (
	"fmt"
	"time"

	"github.com/tidwall/gjson"
)

func aaa(a, b, c string, d, e int) {
	println(fmt.Sprintf("%s, %s, %s, --- %d %d", a, b, c, d, e))
}

type sliceFlag []string

func (f *sliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*f))
}

func (f *sliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

var (
	dirs sliceFlag
)

// xCacheData 可设置超时的缓存字典数据结构
type xCacheData struct {
	Value  interface{}
	Expire time.Time
}

func main() {
	s := "{\"row\":[\"\\灯杆\\单臂\"]}"
	println(s,
		gjson.Parse(s).Get("row").String())
	a := gjson.Parse(s).Get("row").Array()
	println(len(a), gjson.Parse(s).Get("row").IsArray())
}
