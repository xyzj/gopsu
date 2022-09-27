package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/xyzj/gopsu/loopfunc"
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
	loopfunc.LoopFunc(func(params ...interface{}) {
		for {
			time.Sleep(time.Second * 5)
			panic(errors.New("after sleep error"))
		}
	}, "name string", nil)
}
