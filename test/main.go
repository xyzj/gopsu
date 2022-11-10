package main

import (
	"fmt"
	"time"
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
	aa := make(map[string]*xCacheData)
	aa["123"] = &xCacheData{
		Value:  "12334",
		Expire: time.Now().Add(time.Hour),
	}
	aa["456"] = &xCacheData{
		Value:  "12356657",
		Expire: time.Now().Add(time.Hour),
	}
	println("1. ", aa["456"], aa["456"].Expire.GoString())
	v, ok := aa["456"]
	if ok {
		v.Value = "abcddw"
		v.Expire = time.Now().Add(time.Hour * 4)
	}
	println("2. ", v, v.Expire.GoString(), aa["456"].Expire.GoString())
	aa["456"] = v
	println("3. ", aa["456"])
	aa["456"].Expire = time.Now().Add(time.Hour * 4)
	println("4. ", aa["456"])
}
