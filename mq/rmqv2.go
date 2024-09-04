package mq

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/json"
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
	ExchangeAutoDelete bool        // 交换机在不用时是否删除
}

func rmqConnect(opt *RabbitMQOpt, isConsumer bool) (*amqp.Connection, *amqp.Channel, error) {
	var connstr string
	var conn *amqp.Connection
	var channel *amqp.Channel
	var err error
RECONN:
	if opt.TLSConf != nil {
		connstr = fmt.Sprintf("amqps://%s:%s@%s/%s", opt.Username, opt.Passwd, opt.Addr, opt.VHost)
		conn, err = amqp.DialTLS(connstr, opt.TLSConf)
	} else {
		connstr = fmt.Sprintf("amqp://%s:%s@%s/%s", opt.Username, opt.Passwd, opt.Addr, opt.VHost)
		conn, err = amqp.Dial(connstr)
	}
	if err != nil {
		if strings.Contains(err.Error(), "not look like a TLS handshake") {
			opt.TLSConf = nil
			goto RECONN
		}
		if strings.Contains(err.Error(), "Exception (501) Reason") {
			opt.TLSConf = &tls.Config{InsecureSkipVerify: true}
			goto RECONN
		}
		return nil, nil, err
	}
	channel, err = conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	// REEXCHANGE:
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
		conn.Close()
		if strings.Contains(err.Error(), "durable") {
			opt.ExchangeDurable = !opt.ExchangeDurable
			time.Sleep(time.Millisecond * 500)
			goto RECONN
		}
		if strings.Contains(err.Error(), "auto_delete") {
			opt.ExchangeAutoDelete = !opt.ExchangeAutoDelete
			time.Sleep(time.Millisecond * 500)
			goto RECONN
		}
		return nil, nil, err
	}
	if isConsumer {
		_, err = channel.QueueDeclare(
			opt.QueueName,       // name
			opt.QueueDurable,    // durable
			opt.QueueAutoDelete, // delete when unused
			false,               // exclusive
			false,               // no-wait
			amqp.Table{
				amqp.QueueMaxLenArg:     xMaxLength,
				amqp.QueueMessageTTLArg: xMessageTTL,
			}, // arguments
		)
		if err != nil {
			channel.Close()
			conn.Close()
			if strings.Contains(err.Error(), "durable") {
				opt.QueueDurable = !opt.QueueDurable
				time.Sleep(time.Millisecond * 500)
				goto RECONN
			}
			if strings.Contains(err.Error(), "auto_delete") {
				opt.QueueAutoDelete = !opt.QueueAutoDelete
				time.Sleep(time.Millisecond * 500)
				goto RECONN
			}
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
		// 添加一个测试专用的订阅
		channel.QueueBind(opt.QueueName,
			"rmqc.self.test.#",
			opt.ExchangeName,
			false,
			nil)
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
	rcvTime := time.Now().Unix()
	ctxRecv, rcvCancel := context.WithCancel(context.TODO())
	if tself := gopsu.String2Int64(os.Getenv("RMQC_SELF_TEST"), 10) * 60; tself > 0 {
		go loopfunc.LoopFunc(func(params ...interface{}) {
			sendtest := false
			sndOpt := &RabbitMQOpt{
				LogHeader:    "[RMQ-Self]",
				Username:     opt.Username,
				Passwd:       opt.Passwd,
				Addr:         opt.Addr,
				ExchangeName: opt.ExchangeName,
				VHost:        opt.VHost,
			}
			for {
				time.Sleep(time.Minute)
				if time.Now().Unix()-rcvTime <= tself {
					continue
				}
				if sendtest {
					rcvCancel()
					sendtest = false
					continue
				}
				// 指定时间没有数据，开一个生产者进行测试
				snd := NewRMQProducer(sndOpt, logger.NewConsoleLogger())
				snd.Send("rmqc.self.test.recover."+time.Now().Format("2006-01-02.15:04:05.000"), []byte("hello"), time.Second*10)
				time.Sleep(time.Second * 2)
				snd.Close()
				sendtest = true
			}
		}, "rmqc rcv timer", logg.DefaultWriter())
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		rcvTime = time.Now().Unix()
		conn, channel, err := rmqConnect(opt, true)
		if err != nil {
			logg.Error(opt.LogHeader + "connect to " + opt.Addr + " error: " + err.Error())
			panic(err)
		}
		ctxRecv, rcvCancel = context.WithCancel(context.TODO())
		rcvMQ, err := channel.ConsumeWithContext(
			ctxRecv,
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
		logg.System(opt.LogHeader + "Success connect to " + opt.Addr + "; exchange: `" + opt.ExchangeName + "`")
		x = true
		for {
			d := <-rcvMQ
			if d.ContentType == "" && d.DeliveryTag == 0 { // 接收错误，可能服务断开
				x = false
				channel.Close()
				conn.Close()
				logg.Error(opt.LogHeader + "E:Possible service error")
				panic(errors.New(opt.LogHeader + "E:Possible service error," + d.ContentType + "," + strconv.Itoa(int(d.DeliveryTag))))
			}
			logg.Debug(opt.LogHeader + "R:" + d.RoutingKey + " | " + FormatMQBody(d.Body))
			rcvTime = time.Now().Unix()
			func() {
				defer func() {
					if err := recover(); err != nil {
						logg.Error(fmt.Sprintf(opt.LogHeader+"E:calllback error, %+v", errors.WithStack(err.(error))))
					}
				}()
				recvCallback(d.RoutingKey, d.Body)
			}()
		}
	}, opt.LogHeader, logg.DefaultWriter())
	return &x
}

// RMQProducer rmq发送者
type RMQProducer struct {
	sendData  chan *rmqSendData
	ctxClose  context.Context
	sndCancel context.CancelFunc
	ready     bool
}

// Enable rmq发送是否可用
func (r *RMQProducer) Enable() bool {
	return r.ready
}

func (r *RMQProducer) Close() {
	r.sndCancel()
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
	sender := &RMQProducer{
		sendData: make(chan *rmqSendData, 1000),
	}
	sender.ctxClose, sender.sndCancel = context.WithCancel(context.TODO())

	ctxReady, cancel := context.WithTimeout(context.TODO(), time.Second*2)
	go loopfunc.LoopFunc(func(params ...interface{}) {
		conn, channel, err := rmqConnect(opt, false)
		if err != nil {
			logg.Error(opt.LogHeader + "connect error: " + err.Error())
			panic(err)
		}
		logg.System(opt.LogHeader + "Success connect to " + opt.Addr + "; exchange: `" + opt.ExchangeName + "`")
		sender.ready = true
		cancel()
		for {
			select {
			case <-sender.ctxClose.Done():
				sender.ready = false
				logg.Error(opt.LogHeader + "sender close")
				return
			case d := <-sender.sendData:
				ex := strconv.Itoa(int(d.expire.Milliseconds()))
				if ex == "0" {
					ex = "600000"
				}
				err := channel.PublishWithContext(
					context.TODO(),
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
				logg.Debug(opt.LogHeader + "S:" + d.topic + " | " + FormatMQBody(d.body))
			}
		}
	}, opt.LogHeader, logg.DefaultWriter())
	<-ctxReady.Done()
	return sender
}

// FormatMQBody 格式化日志输出
func FormatMQBody(d []byte) string {
	if json.Valid(d) {
		return gopsu.String(d)
	}
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, string(d)))
	// return base64.StdEncoding.EncodeToString(d)
}
