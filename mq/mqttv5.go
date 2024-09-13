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
	"github.com/xyzj/gopsu/cache"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
)

var (
	payloadFormat byte   = 1
	messageExpiry uint32 = 600
	stNotConnect         = false
	ctxClose      context.Context
	funClose      context.CancelFunc
)

var EmptyMQTTClientV5 = &MqttClientV5{
	empty: true,
	st:    &stNotConnect,
}

type mqttMessage struct {
	qos   byte
	body  []byte
	topic string
}

// MqttOpt mqtt 配置
type MqttOpt struct {
	// TLSConf 日志
	Logg logger.Logger
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
	// 是否启用断连消息暂存
	CacheFailed bool
	// 最大缓存消息数量，默认10000
	MaxFailedCache int
}

// MqttClientV5 mqtt客户端 5.0
type MqttClientV5 struct {
	cnf         *MqttOpt
	client      *autopaho.ConnectionManager
	failedCache *cache.AnyCache[*mqttMessage]
	st          *bool
	empty       bool
	ctxCancel   context.CancelFunc
}

// Close close the mqtt client
func (m *MqttClientV5) Close() error {
	if m.empty {
		return nil
	}
	if m.client == nil {
		return nil
	}
	m.failedCache.Close()
	m.st = &stNotConnect
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := m.client.Disconnect(ctx)
	if err != nil {
		return err
	}
	m.ctxCancel()
	return nil
}

// Client return autopaho.ConnectionManager
func (m *MqttClientV5) Client() *autopaho.ConnectionManager {
	if m.empty {
		return nil
	}
	return m.client
}

// IsConnectionOpen 返回在线状态
func (m *MqttClientV5) IsConnectionOpen() bool {
	if m.empty {
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
	if m.empty {
		return nil
	}
	if !*m.st || m.client == nil { // 未连接状态
		if m.cnf.CacheFailed {
			if m.failedCache.Len() < m.cnf.MaxFailedCache {
				m.failedCache.StoreWithExpire(
					time.Now().Format("2006-01-02 15:04:05.999999999"),
					&mqttMessage{
						topic: topic,
						body:  body,
						qos:   qos,
					},
					time.Hour)
			}
		}
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
		m.cnf.Logg.Debug(m.cnf.Name + " Err:" + topic + "|" + err.Error())
		return err
	}
	m.cnf.Logg.Debug(m.cnf.Name + " S:" + topic + "|" + json.String(body))
	return nil
}

// NewMQTTClientV5 创建一个5.0的mqtt client
func NewMQTTClientV5(opt *MqttOpt, recvCallback func(topic string, body []byte)) (*MqttClientV5, error) {
	if opt == nil {
		return EmptyMQTTClientV5, fmt.Errorf("mqtt opt error")
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
	if opt.Logg == nil {
		opt.Logg = &logger.NilLogger{}
	}
	if opt.MaxFailedCache == 0 {
		opt.MaxFailedCache = 10000
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
		return EmptyMQTTClientV5, err
	}
	st := false
	failedCache := cache.NewAnyCache[*mqttMessage](time.Hour)
	conf := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     55,
		CleanStartOnInitialConnection: true,
		TlsCfg:                        opt.TLSConf,
		ConnectRetryDelay:             time.Second * time.Duration(rand.Int31n(30)+30),
		ConnectTimeout:                time.Second * 5,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, c *paho.Connack) {
			st = true
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
			opt.Logg.System(opt.Name + " Success connect to " + opt.Addr)
			// 对失败消息进行补发
			if opt.CacheFailed {
				var err error
				failedCache.ForEach(func(key string, value *mqttMessage) bool {
					ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
					defer cancel()
					err = cm.PublishViaQueue(ctx, &autopaho.QueuePublish{
						Publish: &paho.Publish{
							QoS:     value.qos,
							Topic:   value.topic,
							Payload: value.body,
							Retain:  false,
							Properties: &paho.PublishProperties{
								PayloadFormat: &payloadFormat,
								MessageExpiry: &messageExpiry,
								ContentType:   "text/plain",
							},
						},
					})
					if err != nil {
						opt.Logg.Error(opt.Name + " ReSend `" + value.topic + "` error:" + err.Error())
					}
					return true
				})
			}
		},
		OnConnectError: func(err error) {
			st = false
			opt.Logg.Error(opt.Name + " connect error: " + err.Error())
		},
		ConnectUsername: opt.Username,
		ConnectPassword: []byte(opt.Passwd),
		ClientConfig: paho.ClientConfig{
			ClientID: opt.ClientID, // gopsu.GetRandomString(19, true),
			OnServerDisconnect: func(d *paho.Disconnect) {
				st = false
				if d.ReasonCode == 142 { // client id 重复
					d.Packet().Properties.AssignedClientID += time.Now().Format("_2006-01-02_15:04:05.000000") // "_" + gopsu.GetRandomString(19, true)
					return
				}
				opt.Logg.Error(opt.Name + " server may be down " + strconv.Itoa(int(d.ReasonCode)))
			},
			OnClientError: func(err error) {
				st = false
				opt.Logg.Error(opt.Name + " client error: " + err.Error())
			},
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					opt.Logg.Debug(opt.Name + " R:" + pr.Packet.Topic)
					recvCallback(pr.Packet.Topic, pr.Packet.Payload)
					return true, nil
				},
			},
		},
	}
	ctxClose, funClose = context.WithCancel(context.TODO())
	cm, err := autopaho.NewConnection(ctxClose, conf)
	if err != nil {
		opt.Logg.Error(opt.Name + " new connection error: " + err.Error())
		return EmptyMQTTClientV5, err
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()
	err = cm.AwaitConnection(ctx)
	if err != nil {
		return EmptyMQTTClientV5, err
	}

	return &MqttClientV5{
		client:      cm,
		st:          &st,
		cnf:         opt,
		ctxCancel:   funClose,
		failedCache: failedCache,
	}, nil
}
