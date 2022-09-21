package main

import (
	"fmt"
	"strings"
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
	s := []string{"-d", "-dir=aa", "-d=true", "-aefef", "-d", "true"}
	xss := make([]string, 0)
	idx := 0
	for k, v := range s {
		if v == "-d" || v == "-d=true" {
			idx = k + 1
			continue
		}
		if idx > 0 && idx == k && !strings.HasPrefix(v, "-") {
			continue
		}
		xss = append(xss, v)
	}
	println(strings.Join(xss, " "))
}
