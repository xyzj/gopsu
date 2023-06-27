package main

import (
	"sync"
	"time"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/mq"
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
	opt := &mq.RabbitMQOpt{
		Subscribe:       []string{"test.#"},
		ExchangeName:    "luwak_topic",
		Addr:            "192.168.50.83:5672",
		Username:        "arx7",
		Passwd:          "arbalest",
		VHost:           "",
		QueueName:       "test_" + gopsu.GetRandomString(10, true),
		QueueDurable:    false,
		QueueAutoDelete: true,
	}
	opt2 := &mq.RabbitMQOpt{
		Subscribe:       []string{"test.#"},
		ExchangeName:    "luwak_topic",
		Addr:            "192.168.50.83:5672",
		Username:        "arx7",
		Passwd:          "arbalest",
		VHost:           "",
		QueueName:       "test_" + gopsu.GetRandomString(10, true),
		QueueDurable:    false,
		QueueAutoDelete: true,
	}
	mq.NewRMQConsumer(opt, logger.NewConsoleLogger(), func(topic string, body []byte) {
		println("recv: "+topic, len(body))
	})
	sen := mq.NewRMQProducer(opt2, logger.NewConsoleLogger())
	go func() {
		for {
			time.Sleep(time.Second * 5)
			sen.Send("test.abc", []byte(gopsu.GetRandomString(20, true)), 0)
		}
	}()
	select {}
}
