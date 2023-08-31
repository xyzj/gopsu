// Package mq mqtt 和 rmq 相关功能模块
package mq

import (
	"crypto/tls"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
)

// MqttOpt mqtt 配置
type MqttOpt struct {
	TLSConf   *tls.Config     // tls配置
	Subscribe map[string]byte // 订阅消息，map[topic]qos
	SendTimeo time.Duration   // 发送超时
	ClientID  string
	Addr      string
	Username  string
	Passwd    string
}

// MqttClient mqtt客户端
type MqttClient struct {
	client mqtt.Client
}

func (m *MqttClient) Client() mqtt.Client {
	return m.client
}

func (m *MqttClient) IsConnectionOpen() bool {
	if m.client == nil {
		return false
	}
	return m.client.IsConnectionOpen()
}

func (m *MqttClient) Write(topic string, body []byte) error {
	return m.WriteWithQos(topic, body, 0)
}

func (m *MqttClient) WriteWithQos(topic string, body []byte, qos byte) error {
	if m.client == nil {
		return fmt.Errorf("not connect to the server")
	}
	t := m.client.Publish(topic, qos, false, body)
	t.Wait()
	return t.Error()
}

// NewMQTTClient 创建一个mqtt客户端
func NewMQTTClient(opt *MqttOpt, logg logger.Logger, recvCallback func(topic string, body []byte)) *MqttClient {
	if opt == nil {
		return nil
	}
	if opt.SendTimeo == 0 {
		opt.SendTimeo = time.Second * 5
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
	xopt.SetWriteTimeout(opt.SendTimeo) // 发送3秒超时
	xopt.SetConnectTimeout(time.Second * 10)
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
					logg.Debug("[MQTT] DR:" + msg.Topic() + "; " + json.ToString(msg.Payload()))
					recvCallback(msg.Topic(), msg.Payload())
				})
				doneSub = true
			}
			<-t.C
		}
	}, "[MQTT]", logg.DefaultWriter())
	return &MqttClient{client: client}
}
