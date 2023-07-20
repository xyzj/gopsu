// Package mq mqtt 和 rmq 相关功能模块
package mq

import (
	"crypto/tls"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
)

// MqttOpt mqtt 配置
type MqttOpt struct {
	TLSConf   *tls.Config     // tls配置
	Subscribe map[string]byte // 订阅消息，map[topic]qos
	ClientID  string
	Addr      string
	Username  string
	Passwd    string
}

// NewMQTTClient 创建一个mqtt客户端
func NewMQTTClient(opt *MqttOpt, logg logger.Logger, recvCallback func(topic string, body []byte)) mqtt.Client {
	if opt == nil {
		return nil
	}
	if recvCallback == nil {
		recvCallback = func(topic string, body []byte) {}
	}
	if logg == nil {
		logg = &logger.NilLogger{}
	}
	var needSub = len(opt.Subscribe) > 0
	var doneSub = false
	xopt := mqtt.NewClientOptions()
	xopt.AddBroker("tcp://" + opt.Addr)
	xopt.SetClientID(opt.ClientID + "_" + gopsu.GetRandomString(10, true))
	xopt.SetUsername(opt.Username)
	xopt.SetPassword(opt.Passwd)
	xopt.SetWriteTimeout(time.Second * 3) // 发送3秒超时
	xopt.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logg.Error("[MQTT] connection lost, " + err.Error())
		doneSub = false
	})
	xopt.SetOnConnectHandler(func(client mqtt.Client) {
		logg.System("[MQTT] Success connect to " + opt.Addr)
	})
	client := mqtt.NewClient(xopt)
	go loopfunc.LoopFunc(func(params ...interface{}) {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
		t := time.NewTicker(time.Second * 20)
		for {
			if needSub && !doneSub && client.IsConnectionOpen() {
				client.SubscribeMultiple(opt.Subscribe, func(client mqtt.Client, msg mqtt.Message) {
					defer func() {
						if err := recover(); err != nil {
							logg.Error("[MQTT] " + fmt.Sprintf("%+v", errors.WithStack(err.(error))))
						}
					}()
					recvCallback(msg.Topic(), msg.Payload())
				})
				doneSub = true
			}
			<-t.C
		}
	}, "[MQTT]", logg.DefaultWriter())
	return client
}
