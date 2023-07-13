package mq

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
)

const (
	xMaxLength  = 77777
	xMessageTTL = 600000
)

// RabbitMQOpt rabbitmq 配置
type RabbitMQOpt struct {
	TLSConf            *tls.Config // tls配置
	Subscribe          []string    // 订阅topic
	LogHeader          string      // 日志头
	Addr               string      // 地址
	Username           string      // 用户名
	Passwd             string      // 密码
	VHost              string      // vhost名称
	ExchangeName       string      // 交换机名称
	QueueName          string      // 队列名
	QueueDurable       bool        // 队列是否持久化
	QueueAutoDelete    bool        // 队列在不用时是否删除
	ExchangeDurable    bool        // 交换机是否持久化
	ExchangeAutoDelete bool        //交换机在不用时是否删除
}

func rmqConnect(opt *RabbitMQOpt, isConsumer bool) (*amqp.Connection, *amqp.Channel, error) {
	var connstr string
	var conn *amqp.Connection
	var err error
	if opt.TLSConf != nil {
		connstr = fmt.Sprintf("amqps://%s:%s@%s/%s", opt.Username, opt.Passwd, opt.Addr, opt.VHost)
		conn, err = amqp.DialTLS(connstr, opt.TLSConf)
	} else {
		connstr = fmt.Sprintf("amqp://%s:%s@%s/%s", opt.Username, opt.Passwd, opt.Addr, opt.VHost)
		conn, err = amqp.Dial(connstr)
	}
	if err != nil {
		return nil, nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
REEXCHANGE:
	err = channel.ExchangeDeclare(
		opt.ExchangeName,       // name
		"topic",                // type
		opt.ExchangeDurable,    // durable
		opt.ExchangeAutoDelete, // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		if strings.Contains(err.Error(), "durable") {
			opt.ExchangeDurable = !opt.ExchangeDurable
			time.Sleep(time.Second)
			goto REEXCHANGE
		}
		if strings.Contains(err.Error(), "auto_delete") {
			opt.ExchangeAutoDelete = !opt.ExchangeAutoDelete
			time.Sleep(time.Second)
			goto REEXCHANGE
		}
		conn.Close()
		return nil, nil, err
	}
	if isConsumer {
	REQUEUE:
		_, err = channel.QueueDeclare(
			opt.QueueName,       // name
			opt.QueueDurable,    // durable
			opt.QueueAutoDelete, // delete when unused
			false,               // exclusive
			false,               // no-wait
			amqp.Table{
				"x-max-length":  xMaxLength,
				"x-message-ttl": xMessageTTL,
			}, // arguments
		)
		if err != nil {
			if strings.Contains(err.Error(), "durable") {
				opt.QueueDurable = !opt.QueueDurable
				time.Sleep(time.Second)
				goto REQUEUE
			}
			if strings.Contains(err.Error(), "auto_delete") {
				opt.QueueAutoDelete = !opt.QueueAutoDelete
				time.Sleep(time.Second)
				goto REQUEUE
			}
			channel.Close()
			conn.Close()
			return nil, nil, err
		}
		for _, v := range opt.Subscribe {
			if err := channel.QueueBind(opt.QueueName,
				v,
				opt.ExchangeName,
				false,
				nil); err != nil {
				channel.Close()
				conn.Close()
				return nil, nil, err
			}
		}
	}
	return conn, channel, nil
}

// NewRMQConsumer 创建新的rmq消费者
func NewRMQConsumer(opt *RabbitMQOpt, logg logger.Logger, recvCallback func(topic string, body []byte)) *bool {
	x := false
	if opt == nil {
		return &x
	}
	if logg == nil {
		logg = &logger.NilLogger{}
	}
	if len(opt.Subscribe) == 0 {
		return &x
	}
	if opt.LogHeader == "" {
		opt.LogHeader = "[RMQ-C] "
	}
	if recvCallback == nil {
		recvCallback = func(topic string, body []byte) {}
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		conn, channel, err := rmqConnect(opt, true)
		if err != nil {
			panic(err)
		}
		rcvMQ, err := channel.Consume(
			opt.QueueName,
			"",    // Consumer
			true,  // Auto-Ack
			false, // Exclusive
			false, // No-local
			false, // No-Wait
			nil)   // Args
		if err != nil {
			channel.Close()
			conn.Close()
			panic(err)
		}
		logg.System(opt.LogHeader + "Success connect to " + opt.Addr)
		x = true
		for {
			select {
			case d := <-rcvMQ:
				if d.ContentType == "" && d.DeliveryTag == 0 { // 接收错误，可能服务断开
					x = false
					channel.Close()
					conn.Close()
					panic(errors.New(opt.LogHeader + "E: Possible service error"))
				}
				logg.Debug(opt.LogHeader + "D:" + d.RoutingKey + " | " + FormatMQBody(d.Body))
				func() {
					defer func() {
						if err := recover(); err != nil {
							logg.Error(fmt.Sprintf(opt.LogHeader+"E: calllback error, %+v", errors.WithStack(err.(error))))
						}
					}()
					recvCallback(d.RoutingKey, d.Body)
				}()
			}
		}
	}, "[RMQ-C]", logg.DefaultWriter())
	return &x
}

// RMQProducer rmq发送者
type RMQProducer struct {
	sendData chan *rmqSendData
	ready    bool
}

// Enable rmq发送是否可用
func (r *RMQProducer) Enable() bool {
	return r.ready
}

// Send rmq发送数据
func (r *RMQProducer) Send(topic string, body []byte, expire time.Duration) {
	if !r.ready {
		return
	}

	r.sendData <- &rmqSendData{
		topic:  topic,
		body:   body,
		expire: expire,
	}
}

type rmqSendData struct {
	expire time.Duration
	body   []byte
	topic  string
}

// NewRMQProducer 新的rmq生产者
func NewRMQProducer(opt *RabbitMQOpt, logg logger.Logger) *RMQProducer {
	if opt == nil {
		return nil
	}
	if logg == nil {
		logg = &logger.NilLogger{}
	}
	if opt.LogHeader == "" {
		opt.LogHeader = "[RMQ-P] "
	}
	var sender = &RMQProducer{
		sendData: make(chan *rmqSendData, 1000),
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		conn, channel, err := rmqConnect(opt, false)
		if err != nil {
			panic(err)
		}
		logg.System(opt.LogHeader + "Success connect to " + opt.Addr)
		sender.ready = true
		for {
			select {
			case d := <-sender.sendData:
				ex := strconv.Itoa(int(d.expire.Milliseconds()))
				if ex == "0" {
					ex = "600000"
				}
				err := channel.Publish(
					opt.ExchangeName, // exchange
					d.topic,          // routing key
					true,             // mandatory
					false,            // immediate
					amqp.Publishing{
						ContentType:  "text/plain",
						DeliveryMode: amqp.Persistent,
						Expiration:   ex,
						Timestamp:    time.Now(),
						Body:         d.body,
					},
				)
				if err != nil {
					logg.Error(opt.LogHeader + "E:" + err.Error())
					sender.ready = false
					channel.Close()
					conn.Close()
					panic(err)
				}
				logg.Debug(opt.LogHeader + "D:" + d.topic + " | " + FormatMQBody(d.body))
			}
		}
	}, opt.LogHeader, logg.DefaultWriter())
	return sender
}

// FormatMQBody 格式化日志输出
func FormatMQBody(d []byte) string {
	if json.Valid(d) {
		return gopsu.String(d)
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, gopsu.String(d))
	// return base64.StdEncoding.EncodeToString(d)
}
