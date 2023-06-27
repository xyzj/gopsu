package mq

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	ExchangeName       string      // 交换机名称
	Addr               string      // 地址
	Username           string      // 用户名
	Passwd             string      // 密码
	VHost              string      // vhost名称
	QueueName          string      // 队列名
	QueueDurable       bool        // 队列是否持久化
	QueueAutoDelete    bool        // 队列在不用时是否删除
	ExchangeDurable    bool        // 交换机是否持久化
	ExchangeAutoDelete bool        //交换机在不用时是否删除
}

func rmqConnect(opt *RabbitMQOpt) (*amqp.Connection, *amqp.Channel, error) {
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
		}
		if strings.Contains(err.Error(), "auto_delete") {
			opt.ExchangeAutoDelete = !opt.ExchangeAutoDelete
		}
		conn.Close()
		return nil, nil, err
	}
	return conn, channel, nil
}

// NewRMQConsumer 创建新的rmq消费者
func NewRMQConsumer(opt *RabbitMQOpt, logg logger.Logger, recvCallback func(topic string, body []byte)) {
	if opt == nil {
		return
	}
	if logg == nil {
		logg = &logger.NilLogger{}
	}
	if len(opt.Subscribe) == 0 {
		return
	}
	if recvCallback == nil {
		recvCallback = func(topic string, body []byte) {}
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		conn, channel, err := rmqConnect(opt)
		if err != nil {
			panic(err)
		}
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
			channel.Close()
			conn.Close()
			panic(err)
		}
		for _, v := range opt.Subscribe {
			channel.QueueBind(opt.QueueName,
				v,
				opt.ExchangeName,
				false,
				nil)
		}
		logg.System("[RMQ-C] Success connect to " + opt.Addr)
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
		for {
			select {
			case d := <-rcvMQ:
				if d.ContentType == "" && d.DeliveryTag == 0 { // 接收错误，可能服务断开
					channel.Close()
					conn.Close()
					panic(errors.New("Rcv Err: Possible service error"))
				}
				logg.Debug("[RMQ-C] D: " + d.RoutingKey + " | " + FormatMQBody(d.Body))
				func() {
					defer func() {
						if err := recover(); err != nil {
							logg.Error(fmt.Sprintf("%+v", errors.WithStack(err.(error))))
						}
					}()
					recvCallback(d.RoutingKey, d.Body)
				}()
			}
		}
	}, "[RMQ-C]", logg.DefaultWriter())
}

// RMQProducer rmq发送者
type RMQProducer struct {
	sendData chan *rmqSendData
	ready    bool
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
	var sender = &RMQProducer{
		sendData: make(chan *rmqSendData, 1000),
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		conn, channel, err := rmqConnect(opt)
		if err != nil {
			panic(err)
		}
		logg.System("[RMQ-P] Success connect to " + opt.Addr)
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
					logg.Error("[RMQ-P] E:" + err.Error())
					channel.Close()
					conn.Close()
					sender.ready = false
					panic(err)
				} else {
					logg.Debug("[RMQ-P] D: " + d.topic + " | " + FormatMQBody(d.body))
				}
			}
		}
	}, "[RMQ-P]", logg.DefaultWriter())
	return sender
}

// FormatMQBody 格式化日志输出
func FormatMQBody(d []byte) string {
	if json.Valid(d) {
		return gopsu.String(d)
	}
	return base64.StdEncoding.EncodeToString(d)
}
