package mq

// broker: https://github.com/nanomq/nanomq/releases/tag/0.21.6

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/logger"
)

var (
	payloadFormat byte   = 1
	messageExpiry uint32 = 3600
)

// MqttClientV5 mqtt客户端 5.0
type MqttClientV5 struct {
	client *autopaho.ConnectionManager
	st     *bool
}

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
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	return m.client.PublishViaQueue(ctx, &autopaho.QueuePublish{
		Publish: &paho.Publish{
			QoS:     qos,
			Topic:   topic,
			Payload: body,
			Properties: &paho.PublishProperties{
				PayloadFormat: &payloadFormat,
				MessageExpiry: &messageExpiry,
			},
		},
	})
	// _, err := m.client.Publish(ctx, &paho.Publish{
	// 	QoS:     qos,
	// 	Topic:   topic,
	// 	Payload: body,
	// 	Properties: &paho.PublishProperties{
	// 		PayloadFormat: &payloadFormat,
	// 		MessageExpiry: &messageExpiry,
	// 	},
	// })
	// return err
}

// NewMQTTClientV5 创建一个5.0的mqtt client
func NewMQTTClientV5(opt *MqttOpt, logg logger.Logger, recvCallback func(topic string, body []byte)) (*MqttClientV5, error) {
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
	if logg == nil {
		logg = &logger.NilLogger{}
	}
	opt.ClientID += "_" + gopsu.GetRandomString(20, true)
	if len(opt.ClientID) > 22 {
		opt.ClientID = opt.ClientID[:22]
	}
	if !strings.HasPrefix(opt.Addr, "tls://") && !strings.HasPrefix(opt.Addr, "mqtt://") && !strings.HasPrefix(opt.Addr, "mqtts://") {
		opt.Addr = "mqtt://" + opt.Addr
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
		SessionExpiryInterval:         300,
		TlsCfg:                        opt.TLSConf,
		ConnectRetryDelay:             time.Minute,
		ConnectTimeout:                time.Second * 10,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, c *paho.Connack) {
			logg.System(opt.Name + " Success connect to " + opt.Addr)
			st = true
		},
		OnConnectError: func(err error) {
			logg.Error(opt.Name + " connection lost, " + err.Error())
			st = false
		},
		ConnectUsername: opt.Username,
		ConnectPassword: []byte(opt.Passwd),
		ClientConfig: paho.ClientConfig{
			ClientID: opt.ClientID,
			OnServerDisconnect: func(d *paho.Disconnect) {
				logg.Error(opt.Name + " server may be down " + d.Properties.ReasonString)
				st = false
			},
			OnClientError: func(err error) {
				logg.Error(opt.Name + " client error: " + err.Error())
				st = false
			},
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					logg.Debug(opt.Name + " DR:" + gopsu.String(pr.Packet.Payload))
					st = false
					recvCallback(pr.Packet.Topic, pr.Packet.Payload)
					return true, nil
				},
			},
		},
	}
	cm, err := autopaho.NewConnection(context.Background(), conf)
	if err != nil {
		logg.Error(opt.Name + " connect to server error: " + err.Error())
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err = cm.AwaitConnection(ctx); err != nil {
		logg.Error(opt.Name + " connect to server timeout: " + err.Error())
		return nil, err
	}
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

	return &MqttClientV5{
		client: cm,
		st:     &st,
	}, nil
}
