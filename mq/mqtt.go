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
	// tls配置，默认为 InsecureSkipVerify: true
	TLSConf *tls.Config
	// 订阅消息，map[topic]qos
	Subscribe map[string]byte
	// 发送超时
	SendTimeo time.Duration
	// ClientID 客户端标示，会添加随机字符串尾巴，最大22个字符
	ClientID string
	// 服务端ip:port
	Addr string
	// 登录用户名
	Username string
	// 登录密码
	Passwd string
	// 日志前缀，默认 [MQTT]
	Name string
}

// MqttClient mqtt客户端
type MqttClient struct {
	client mqtt.Client
}

func (m *MqttClient) Client() mqtt.Client {
	return m.client
}

// IsConnectionOpen 返回在线状态
func (m *MqttClient) IsConnectionOpen() bool {
	if m.client == nil {
		return false
	}
	return m.client.IsConnectionOpen()
}

// Write 以qos0发送消息
func (m *MqttClient) Write(topic string, body []byte) error {
	return m.WriteWithQos(topic, body, 0)
}

// WriteWithQos 发送消息，可自定义qos
func (m *MqttClient) WriteWithQos(topic string, body []byte, qos byte) error {
	if m.client == nil {
		return fmt.Errorf("not connect to the server")
	}
	t := m.client.Publish(topic, qos, false, body)
	t.Wait()
	return t.Error()
}

// NewMQTTClient 创建一个mqtt客户端 3.11
func NewMQTTClient(opt *MqttOpt, logg logger.Logger, recvCallback func(topic string, body []byte)) (*MqttClient, error) {
	if opt == nil {
		return nil, fmt.Errorf("mqtt opt error")
	}
	if opt.SendTimeo == 0 {
		opt.SendTimeo = time.Second * 5
	}
	if opt.Name == "" {
		opt.Name = "[MQTT]"
	}
	if opt.TLSConf == nil {
		opt.TLSConf = &tls.Config{InsecureSkipVerify: true}
	}

	if recvCallback == nil {
		recvCallback = func(topic string, body []byte) {}
	}
	if logg == nil {
		logg = &logger.NilLogger{}
	}

	if opt.ClientID == "" {
		opt.ClientID += "_" + gopsu.GetRandomString(20, true)
	}
	if len(opt.ClientID) > 22 {
		opt.ClientID = opt.ClientID[:22]
	}
	var needSub = len(opt.Subscribe) > 0
	var doneSub = false
	xopt := mqtt.NewClientOptions()
	xopt.AddBroker("tcp://" + opt.Addr)
	xopt.SetClientID(opt.ClientID)
	xopt.SetUsername(opt.Username)
	xopt.SetPassword(opt.Passwd)
	xopt.SetTLSConfig(opt.TLSConf)
	xopt.SetWriteTimeout(opt.SendTimeo) // 发送3秒超时
	xopt.SetConnectTimeout(time.Second * 10)
	xopt.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logg.Error(opt.Name + " connection lost, " + err.Error())
		doneSub = false
	})
	xopt.SetOnConnectHandler(func(client mqtt.Client) {
		logg.System(opt.Name + " Success connect to " + opt.Addr)
	})
	client := mqtt.NewClient(xopt)
	go loopfunc.LoopFunc(func(params ...interface{}) {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			logg.Error(opt.Name + " " + token.Error().Error())
			panic(token.Error())
		}
		for {
			if needSub && !doneSub && client.IsConnectionOpen() {
				client.SubscribeMultiple(opt.Subscribe, func(client mqtt.Client, msg mqtt.Message) {
					defer func() {
						if err := recover(); err != nil {
							logg.Error(opt.Name + fmt.Sprintf(" %+v", errors.WithStack(err.(error))))
						}
					}()
					logg.Debug(opt.Name + " DR:" + msg.Topic() + "; " + json.String(msg.Payload()))
					recvCallback(msg.Topic(), msg.Payload())
				})
				doneSub = true
			}
			time.Sleep(time.Second * 20)
		}
	}, opt.Name, logg.DefaultWriter())
	return &MqttClient{client: client}, nil
}
