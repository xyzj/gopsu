package main

import (
	"encoding/json"
	"flag"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/gocmd"
	"github.com/xyzj/gopsu/pathtool"
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
func test(a bool, b ...string) {
	if len(b) == 0 {
		println("no b")
	} else {
		if b[0] == "" {
			println("nadadadf")
		} else {
			println("123123123")
		}
	}
	if a {
		defer println("defer")
	}
	println("done")
}

var (
	conf = flag.String("conf", "", "usage")
)

func main() {
	gocmd.DefaultProgram(&gocmd.Info{
		Title:    "a test program",
		Descript: "askdfjl1klk3h1k2j3o1jo3",
		Ver:      gocmd.VersionInfo("test", "0.0.1", "1.20", "", "linux", "xy"),
	}).Execute()
	println("---- conf is", *conf)
	for {
		time.Sleep(time.Second * 5)
		println(pathtool.GetExecDir())
	}
}

var (
	georep = strings.NewReplacer("(", "", ")", "", "POINT ", "", "POLYGON ", "", "LINESTRING ", "") // 经纬度字符串处理替换器
)

func text2Geo(s string) []*assetGeo {
	geostr := strings.Split(georep.Replace(s), ", ")
	gp := make([]*assetGeo, 0)
	for _, v := range geostr {
		vv := strings.Split(v, " ")
		gp = append(gp, &assetGeo{
			Lng: gopsu.String2Float64(vv[0]),
			Lat: gopsu.String2Float64(vv[1]),
		})
	}
	return gp
}

type assetGeo struct {
	Lng  float64 `json:"lng"`
	Lat  float64 `json:"lat"`
	Name string  `json:"aid,omitempty"`
}
