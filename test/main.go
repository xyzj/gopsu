package main

import (
	"encoding/json"
	"flag"
	"strings"
	"sync"
	"unicode"

	"github.com/xyzj/gopsu"
	config "github.com/xyzj/gopsu/confile"
)

var (
	version     = "0.0.0"
	goVersion   = ""
	buildDate   = ""
	platform    = ""
	author      = "Xu Yuan"
	programName = "Asset Data Center"
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
	conf  = flag.String("conf", "", "usage")
	conf1 = flag.String("conf1", "", "usage")
	conf2 = flag.String("conf2", "", "usage")
)

func mqttcb(topic string, body []byte) {
	println("---", topic, string(body))
}

type aaa struct {
	Username string          `json:"username" yaml:"username"`
	Password gopsu.PwdString `json:"pwd" yaml:"pwd"`
}

func main() {
	conf := config.NewConfig("test.conf") // 创建/读取配置文件
	println(conf.Print())                 //  查看所有配置项
	println(conf.GetItem("root_path"))    // 读取一个配置项的值
	conf.GetDefault(&config.Item{         // 尝试读取一个配置项的值，当配置项不存在时，添加当前配置项
		Key:          "zzzzzz_path",
		Value:        "asdfaldjlasjfd",
		EncryptValue: true, // 保存时需要将value加密
		Comment:      "1234ksdfkjhasdfh",
	})
	println(conf.GetItem("zzzzzz_path").TryDecode()) // 读取配置项，并解密值
	conf.ToFile()                                    // 保存到文件
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
