package main

import (
	"encoding/json"
	"strings"
	"sync"
	"unicode"

	"github.com/xyzj/gopsu"
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

func FormatMQBody(d []byte) string {
	if json.Valid(d) {
		return gopsu.String(d)
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, gopsu.String(d))
	// return base64.StdEncoding.EncodeToString(d)
}
func main() {
	println(FormatMQBody([]byte("skjhdf23ldsfhasdf")))
}
