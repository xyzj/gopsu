package mq

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
)

const (
	queueMaxLength = 77777
)

// RabbitMQData rabbit-mq data send struct
type RabbitMQData struct {
	RoutingKey string
	Data       *amqp.Publishing
}

// Session rmq session
type Session struct {
	name         string
	logger       gopsu.Logger
	connection   *amqp.Connection
	channel      *amqp.Channel
	done         chan bool
	closeMe      bool
	debug        bool
	connStr      string
	addr         string
	routingKeys  sync.Map    // 过滤器
	queueName    string      // 队列名
	queueDurable bool        // 队列是否持久化
	queueDelete  bool        // 队列在不用时是否删除
	sessnType    string      // consumer or producer
	tlsConf      *tls.Config // tls配置
}

// NewConsumer 初始化消费者实例
// exchangename,connstr,queuename,durable,autodel,debug
func NewConsumer(name, connstr, queuename string, durable, autodel, debug bool) *Session {
	sessn := &Session{
		sessnType:    "consumer",
		name:         name,
		connStr:      connstr,
		debug:        debug,
		done:         make(chan bool),
		queueName:    queuename,
		queueDurable: durable,
		queueDelete:  autodel,
		closeMe:      false,
		logger:       &gopsu.NilLogger{},
	}
	sessn.addr = strings.Split(connstr, "@")[1]
	return sessn
}

// NewProducer 初始化生产者实例
func NewProducer(name, connstr string, debug bool) *Session {
	sessn := &Session{
		sessnType: "producer",
		name:      name,
		connStr:   connstr,
		debug:     debug,
		done:      make(chan bool),
		logger:    &gopsu.NilLogger{},
	}
	sessn.addr = strings.Split(connstr, "@")[1]
	return sessn
}

// Start Start
func (sessn *Session) Start() bool {
	sessn.handleReconnect()
	return sessn.WaitReady(5)
	// if sessn.connect() {
	// 	switch sessn.sessnType {
	// 	case "consumer":
	// 		sessn.initConsumer()
	// 	case "producer":
	// 		sessn.initProducer()
	// 	}
	// 	return sessn.WaitReady(5)
	// }
	// return false
}

// StartTLS 使用ssl连接
func (sessn *Session) StartTLS(t *tls.Config) bool {
	sessn.tlsConf = t
	return sessn.Start()
}

// SetLogger SetLogger
func (sessn *Session) SetLogger(l gopsu.Logger) {
	sessn.logger = l
}

// handleReconnect 维护连接
func (sessn *Session) handleReconnect() {
	defer func() {
		if err := recover(); err != nil {
			sessn.logger.Error(errors.WithStack(err.(error)).Error())
		}
	}()
	if sessn.connect() {
		switch sessn.sessnType {
		case "consumer":
			sessn.initConsumer()
		case "producer":
			sessn.initProducer()
		}
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				println(sessn.sessnType + " crash" + errors.WithStack(err.(error)).Error())
				sessn.logger.Error(errors.WithStack(err.(error)).Error())
			}
		}()
		for {
			if sessn.closeMe {
				break
			}
			select {
			case <-sessn.done:
				sessn.closeMe = true
				sessn.channel.Close()
				sessn.connection.Close()
				sessn.connection = nil
			case <-time.After(7 * time.Second):
				if sessn.IsReady() {
					continue
				}
				if sessn.connect() {
					switch sessn.sessnType {
					case "consumer":
						sessn.initConsumer()
					case "producer":
						sessn.initProducer()
					}
				}
			}
		}
	}()
}

// connect 建立连接
func (sessn *Session) connect() bool {
	if sessn.IsReady() {
		return true
	}
	var exAutoDel = false
CONN:
	// sessn.logger.Warning("Attempting to connect to " + sessn.addr)
	var err error
	if sessn.tlsConf == nil {
		sessn.connection, err = amqp.Dial(sessn.connStr)
	} else {
		sessn.connection, err = amqp.DialTLS(sessn.connStr, sessn.tlsConf)
	}

	if err != nil {
		sessn.logger.Error("Failed connnect to " + sessn.addr + "|" + err.Error())
		return false
	}
	sessn.channel, err = sessn.connection.Channel()
	if err != nil {
		sessn.logger.Error("Failed open channel: " + err.Error())
		return false
	}
	err = sessn.channel.ExchangeDeclare(
		sessn.name, // name
		"topic",    // type
		true,       // durable
		exAutoDel,  // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
	if err == nil {
		sessn.logger.System("Success connect to " + sessn.addr)
		return true
	}
	if strings.Contains(err.Error(), "auto_delete") && !exAutoDel {
		exAutoDel = true
		goto CONN
	}
	sessn.logger.Error("Failed declare exchange: " + err.Error())
	return false
}

// IsReady 是否就绪
func (sessn *Session) IsReady() bool {
	if sessn.connection == nil {
		return false
	}
	return !sessn.connection.IsClosed()
}

// WaitReady 等待就绪，0-默认超时5s
func (sessn *Session) WaitReady(second int) bool {
	if second == 0 {
		second = 5
	}
	time.Sleep(time.Second)
	tc := time.NewTicker(time.Second * time.Duration(second))
	for {
		select {
		case <-time.After(time.Millisecond * 100):
			if sessn.IsReady() {
				return true
			}
		case <-tc.C:
			if sessn.IsReady() {
				return true
			}
			return false
		}
	}
}

// Close 关闭
func (sessn *Session) Close() {
	sessn.done <- true
}

func (sessn *Session) initConsumer() {
	defer func() {
		if err := recover(); err != nil {
			sessn.logger.Error("Consumer core error: " + errors.WithStack(err.(error)).Error())
		}
	}()
	var err error
	_, err = sessn.channel.QueueDeclare(
		sessn.queueName,    // name
		sessn.queueDurable, // durable
		sessn.queueDelete,  // delete when unused
		false,              // exclusive
		false,              // no-wait
		amqp.Table{
			"x-max-length": queueMaxLength,
		}, // arguments
	)
	if err != nil {
		sessn.logger.Error("Failed create queue " + sessn.queueName + ": " + err.Error())
		return
	}
	sessn.routingKeys.Range(func(k, v interface{}) bool {
		sessn.channel.QueueBind(sessn.queueName, k.(string), sessn.name, false, nil)
		return true
	})
}

// Recv 接收消息
func (sessn *Session) Recv() (<-chan amqp.Delivery, error) {
	if !sessn.IsReady() {
		return nil, fmt.Errorf("not connected")
	}
	c, err := sessn.channel.Consume(
		sessn.queueName,
		"",    // Consumer
		true,  // Auto-Ack
		false, // Exclusive
		false, // No-local
		false, // No-Wait
		nil,   // Args
	)
	if err != nil {
		sessn.channel.Close()
		sessn.connection.Close()
		// sessn.connection = nil
		return nil, err
	}
	return c, nil
}

// BindKey 绑定过滤器
func (sessn *Session) BindKey(k ...string) error {
	for _, v := range k {
		if gopsu.TrimString(v) == "" {
			continue
		}
		sessn.routingKeys.Store(v, "")
	}
	if sessn.IsReady() {
		var err error
		var s = make([]string, 0)
		for _, v := range k {
			if gopsu.TrimString(v) == "" {
				continue
			}
			err = sessn.channel.QueueBind(sessn.queueName, v, sessn.name, false, nil)
			if err != nil {
				s = append(s, v)
			}
		}
		if len(s) > 0 {
			return fmt.Errorf(strings.Join(s, ",") + " bind error:" + err.Error())
		}
		return nil
	}
	return fmt.Errorf("failed bind key, channel not ready")
}

// ClearQueue 清空队列
func (sessn *Session) ClearQueue() {
	if sessn.IsReady() {
		sessn.channel.QueuePurge(sessn.queueName, true)
	}
}

// UnBindKey 解绑过滤器
func (sessn *Session) UnBindKey(k ...string) error {
	for _, v := range k {
		if gopsu.TrimString(v) == "" {
			continue
		}
		sessn.routingKeys.Delete(v)
	}
	if sessn.IsReady() {
		var err error
		var s = make([]string, 0)
		for _, v := range k {
			if gopsu.TrimString(v) == "" {
				continue
			}
			err = sessn.channel.QueueUnbind(sessn.queueName, v, sessn.name, nil)
			if err != nil {
				s = append(s, v)
				continue
			}
		}
		if len(s) > 0 {
			return fmt.Errorf(strings.Join(s, ",") + " unbind error:" + err.Error())
		}
		return nil
	}
	return fmt.Errorf("failed Unbind key, channel not ready")
}

func (sessn *Session) initProducer() {
}

// Send 发送数据,默认数据有效期10分钟
func (sessn *Session) Send(f string, d []byte) error {
	return sessn.SendCustom(&RabbitMQData{
		RoutingKey: f,
		Data: &amqp.Publishing{
			ContentType:  "text/plain",
			DeliveryMode: amqp.Persistent,
			Expiration:   "600000",
			Timestamp:    time.Now(),
			Body:         d,
		},
	})
}

// SendCustom 自定义发送参数
//
// amqp.Publishing{
// 	ContentType:  "text/plain",
// 	DeliveryMode: amqp.Persistent,
// 	Expiration:   "300000",
// 	Timestamp:    time.Now(),
// 	Body:         []byte("abcd"),
// },
func (sessn *Session) SendCustom(d *RabbitMQData) error {
	if !sessn.IsReady() {
		return fmt.Errorf("MQ Producer not ready")
	}
	// go func() {
	defer func() {
		if err := recover(); err != nil {
			sessn.logger.Error("SndCrash:" + sessn.addr + "|" + err.(error).Error())
		}
	}()
	err := sessn.channel.Publish(
		sessn.name,   // exchange
		d.RoutingKey, // routing key
		false,        // mandatory
		false,        // immediate
		*d.Data,
	)
	if err != nil {
		sessn.channel.Close()
		sessn.connection.Close()
		// sessn.connection = nil
	}
	return err
}

// FormatMQBody 格式化日志输出
func FormatMQBody(d []byte) string {
	if gjson.ParseBytes(d).Exists() {
		return gopsu.String(d)
	}
	return base64.StdEncoding.EncodeToString(d)
}
