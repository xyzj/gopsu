package main

import (
	"fmt"
	"sync"

	"github.com/xyzj/gopsu/mapfx"
)

// 结构定义
// 设备型号信息
type devmod struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Sys    string `json:"-"`
	Remark string `json:"remark,omitempty"`
	pinyin string
}

func (d devmod) DoNoting() {
}

type BaseMap struct {
	sync.RWMutex
	data map[string]string
}

func main() {
	a := mapfx.NewStructMap[int64, devmod]()
	a.Store(1273678123, &devmod{
		ID:   "23412",
		Name: "123123",
	})
	println(a.HasPrefix("127"))
	println(a.HasPrefix("27"))

	b := BaseMap{
		// sync.RWMutex{},
		data: make(map[string]string),
	}
	println(fmt.Sprintf("%+v", b))
}
