package mq

// broker: https://github.com/nanomq/nanomq/releases/tag/0.21.6

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/logger"
)

var (
	payloadFormat byte   = 1
	messageExpiry uint32 = 600
)

// MqttClientV5 mqtt客户端 5.0
type MqttClientV5 struct {
	opt    *MqttOpt
	client *autopaho.ConnectionManager
	st     *bool
}

// Close close the mqtt client
func (m *MqttClientV5) Close() error {
	if m.client == nil {
		return fmt.Errorf("not connect to the server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	return m.client.Disconnect(ctx)
}

// Client return autopaho.ConnectionManager
func (m *MqttClientV5) Client() *autopaho.ConnectionManager {
	return m.client
}

// IsConnectionOpen 返回在线状态
func (m *MqttClientV5) IsConnectionOpen() bool {
	if m.client == nil {
		return false
	}
	return *m.st
}

// Write 以qos0发送消息
func (m *MqttClientV5) Write(topic string, body []byte) error {
	return m.WriteWithQos(topic, body, 0)
}

// WriteWithQos 发送消息，可自定义qos
func (m *MqttClientV5) WriteWithQos(topic string, body []byte, qos byte) error {
	if m.client == nil {
		return fmt.Errorf("not connect to the server")
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()
	err := m.client.PublishViaQueue(ctx, &autopaho.QueuePublish{
		Publish: &paho.Publish{
			QoS:     qos,
			Topic:   topic,
			Payload: body,
			Retain:  false,
			Properties: &paho.PublishProperties{
				PayloadFormat: &payloadFormat,
				MessageExpiry: &messageExpiry,
				ContentType:   "text/plain",
			},
		},
	})
	if err != nil {
		m.opt.Logg.Debug(m.opt.Name + " DSErr:" + topic + "|" + err.Error())
		return err
	}
	m.opt.Logg.Debug(m.opt.Name + " DS:" + topic)
	return nil
}

// NewMQTTClientV5 创建一个5.0的mqtt client
func NewMQTTClientV5(opt *MqttOpt, recvCallback func(topic string, body []byte)) (*MqttClientV5, error) {
	if opt == nil {
		return nil, fmt.Errorf("mqtt opt error")
	}
	if opt.SendTimeo == 0 {
		opt.SendTimeo = time.Second * 5
	}
	if opt.Name == "" {
		opt.Name = "[MQTTv5]"
	}
	if opt.TLSConf == nil {
		opt.TLSConf = &tls.Config{InsecureSkipVerify: true}
	}

	if recvCallback == nil {
		recvCallback = func(topic string, body []byte) {}
	}
	if opt.Logg == nil {
		opt.Logg = &logger.NilLogger{}
	}
	if !strings.Contains(opt.Addr, "://") {
		switch {
		case strings.Contains(opt.Addr, ":1881"):
			opt.Addr = "tls://" + opt.Addr
		default: // case strings.Contains(opt.Addr,":1883"):
			opt.Addr = "mqtt://" + opt.Addr
		}
	}
	u, err := url.Parse(opt.Addr)
	if err != nil {
		return nil, err
	}
	st := false
	conf := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     40,
		CleanStartOnInitialConnection: false,
		TlsCfg:                        opt.TLSConf,
		ConnectRetryDelay:             time.Second * time.Duration(rand.Int31n(30)+30),
		ConnectTimeout:                time.Second * 5,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, c *paho.Connack) {
			opt.Logg.System(opt.Name + " Success connect to " + opt.Addr)
			if len(opt.Subscribe) > 0 {
				x := make([]paho.SubscribeOptions, 0, len(opt.Subscribe))
				for k, v := range opt.Subscribe {
					x = append(x, paho.SubscribeOptions{
						Topic: k,
						QoS:   v,
					})
				}
				cm.Subscribe(context.Background(), &paho.Subscribe{
					Subscriptions: x,
				})
			}
			st = true
		},
		OnConnectError: func(err error) {
			opt.Logg.Error(opt.Name + " connect error: " + err.Error())
			st = false
		},
		ConnectUsername: opt.Username,
		ConnectPassword: []byte(opt.Passwd),
		ClientConfig: paho.ClientConfig{
			ClientID: opt.ClientID + "_" + gopsu.GetRandomString(19, true),
			OnServerDisconnect: func(d *paho.Disconnect) {
				st = false
				if d.ReasonCode == 142 { // client id 重复
					d.Packet().Properties.AssignedClientID += "__" + gopsu.GetRandomString(19, true)
					return
				}
				opt.Logg.Error(opt.Name + " server may be down " + strconv.Itoa(int(d.ReasonCode)))
			},
			OnClientError: func(err error) {
				opt.Logg.Error(opt.Name + " client error: " + err.Error())
				st = false
			},
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					opt.Logg.Debug(opt.Name + " DR:" + pr.Packet.Topic)
					recvCallback(pr.Packet.Topic, pr.Packet.Payload)
					return true, nil
				},
			},
		},
	}
	cm, err := autopaho.NewConnection(context.Background(), conf)
	if err != nil {
		opt.Logg.Error(opt.Name + " connect to server error: " + err.Error())
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()
	err = cm.AwaitConnection(ctx)
	if err != nil {
		return nil, err
	}

	return &MqttClientV5{
		client: cm,
		st:     &st,
		opt:    opt,
	}, nil
}
