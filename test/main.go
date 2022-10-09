package main

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"
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

func main() {
	s, _ := sjson.Set("", "a.-1", "value interface{}")
	s, _ = sjson.Set(s, "a.-1", "value interface{}")
	s, _ = sjson.Set(s, "a.-1", "value interface{}")
	println(s)
	println(gjson.Parse(s).Get("a").String())
}
