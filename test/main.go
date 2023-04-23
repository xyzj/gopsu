package main

import (
	"fmt"

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

func main() {
	a := mapfx.NewSliceMap[string]()
	a.Store("key", []string{"sdf"})
	a.StoreItem("key", "sdf")
	a.StoreItem("key", "123")
	v, _ := a.Load("key")
	println(fmt.Sprintf("-----------------  %+v", v), v)
	v = append(v, "986")
	v2, _ := a.Load("key")
	println(fmt.Sprintf("-----------------  %+v", v), v)
	println(fmt.Sprintf("-----------------  %+v", v2), v2)
}
