package main

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/mq"
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
	Username string           `json:"username" yaml:"username"`
	Password config.PwdString `json:"pwd" yaml:"pwd"`
}

type serviceParams struct {
	Params     []string `yaml:"params"`
	Exec       string   `yaml:"exec"`
	Enable     bool     `yaml:"enable"`
	manualStop bool     `yaml:"-"`
}

func main() {
	os.Setenv("RMQC_SELF_TEST", "2")
	// go func() {
	// 	snd := mq.NewRMQProducer(&mq.RabbitMQOpt{
	// 		Addr:         "192.168.50.83:5672",
	// 		Username:     "arx7",
	// 		Passwd:       "arbalest",
	// 		ExchangeName: "luwak_topic",
	// 	}, logger.NewConsoleLogger())
	// 	for {
	// 		time.Sleep(time.Second * 3)
	// 		snd.Send("rmqc.self.test", []byte(gopsu.GetRandomString(50, true)), time.Second)
	// 	}
	// }()
	go func() {
		mq.NewRMQConsumer(&mq.RabbitMQOpt{
			Addr:            "192.168.50.83:5672",
			Username:        "arx7",
			Passwd:          "arbalest",
			ExchangeName:    "luwak_topic",
			Subscribe:       []string{"test.#"},
			QueueName:       "test_qqqq",
			QueueAutoDelete: true,
		}, logger.NewConsoleLogger(), func(topic string, body []byte) { println(topic, body) })
	}()
	select {}
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
