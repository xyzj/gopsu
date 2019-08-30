package mq

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/streadway/amqp"
	"github.com/xyzj/gopsu"
)

const (
	queueMaxLength = 70000
)

// RabbitMQData rabbit-mq data send struct
type RabbitMQData struct {
	RoutingKey string
	Data       *amqp.Publishing
}

// Session rmq session
type Session struct {
	name          string
	logger        *io.Writer
	loggerLevel   int
	connection    *amqp.Connection
	channel       *amqp.Channel
	done          chan bool
	closeMe       bool
	isReady       bool
	debug         bool
	connStr       string
	addr          string
	queue         amqp.Queue
	routingKeys   []string           // 过滤器
	queueName     string             // 队列名
	queueDurable  bool               // 队列是否持久化
	queueDelete   bool               // 队列在不用时是否删除
	queueDelivery chan amqp.Delivery // 消息
	sessnType     string             // consumer or producer
}

// NewConsumer 初始化消费者实例
// exchangename,connstr,queuename,logger,durable,autodel,debug
func NewConsumer(name, connstr, queuename string, durable, autodel, debug bool) *Session {
	sessn := &Session{
		sessnType:     "consumer",
		name:          name,
		connStr:       connstr,
		debug:         debug,
		done:          make(chan bool),
		queueName:     queuename,
		queueDurable:  durable,
		queueDelete:   autodel,
		queueDelivery: make(chan amqp.Delivery),
		closeMe:       false,
	}
	sessn.addr = strings.Split(connstr, "@")[1]
	// go sessn.handleReconnect("consumer")
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
	}
	sessn.addr = strings.Split(connstr, "@")[1]
	// go sessn.handleReconnect("producer")
	return sessn
}

// Start Start
func (sessn *Session) Start() {
	sessn.handleReconnect(sessn.sessnType)
}

// SetLogger SetLogger
func (sessn *Session) SetLogger(w *io.Writer, l int) {
	sessn.logger = w
	sessn.loggerLevel = l
}

func (sessn *Session) writeLog(s string, l int) {
	s = fmt.Sprintf("%v [%02d] [MQ] %s", time.Now().Format(gopsu.LogTimeFormat), l, s)
	if sessn.logger == nil {
		if sessn.debug {
			println(s)
		}
	} else {
		if l >= sessn.loggerLevel && l < 90 {
			fmt.Fprintln(*sessn.logger, s)
		} else if l == 90 {
			println(s)
		}
	}
}

// handleReconnect 维护连接
func (sessn *Session) handleReconnect(t string) {
	defer func() {
		if err := recover(); err != nil {
			sessn.writeLog(err.(error).Error(), 40)
		}
	}()
	if sessn.connect() {
		switch t {
		case "consumer":
			sessn.initConsumer()
		case "producer":
			sessn.initProducer()
		}
	}
	for {
		if sessn.closeMe {
			break
		}
		select {
		case <-sessn.done:
			sessn.closeMe = true
			sessn.channel.Close()
			sessn.connection.Close()
			_, ok := <-sessn.queueDelivery
			if ok {
			}
		case <-time.After(10 * time.Second):
			if sessn.isReady {
				continue
			}
			if sessn.connect() {
				switch t {
				case "consumer":
					sessn.initConsumer()
				case "producer":
					sessn.initProducer()
				}
			}
		}
	}
}

// connect 建立连接
func (sessn *Session) connect() bool {
	sessn.isReady = false
	sessn.writeLog("Attempting to connect to "+sessn.addr, 30)
	conn, err := amqp.Dial(sessn.connStr)

	if err != nil {
		sessn.writeLog("Failed to connnect to "+sessn.addr, 40)
		return false
	}
	sessn.connection = conn

	ch, err := conn.Channel()
	if err != nil {
		sessn.writeLog("Failed to open channel: "+err.Error(), 40)
		return false
	}
	sessn.channel = ch

	err = sessn.channel.ExchangeDeclare(
		sessn.name, // name
		"topic",    // type
		true,       // durable
		true,       // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		sessn.writeLog("Failed to declare exchange: "+err.Error(), 40)
		return false
	}

	sessn.writeLog("Success to connect to "+sessn.addr, 90)

	sessn.isReady = true
	return true
}

// IsReady 是否就绪
func (sessn *Session) IsReady() bool {
	return sessn.isReady
}

// WaitReady 等待就绪，0-默认超时5s
func (sessn *Session) WaitReady(second int) bool {
	if second == 0 {
		second = 5
	}
	for {
		select {
		case <-time.After(time.Millisecond * 10):
			if sessn.isReady {
				return true
			}
		case <-time.Tick(time.Duration(second) * time.Second):
			if sessn.isReady {
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
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		sessn.isReady = false
	// 		sessn.logError("Consumer core error: " + errors.WithStack(err.(error)).Error())
	// 	}
	// }()
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
		sessn.writeLog("Failed to create queue "+sessn.queueName+": "+err.Error(), 40)
		sessn.isReady = false
		return
	}

	// delivery, err := sessn.channel.Consume(sessn.name, // queue
	// 	"",    // consumer
	// 	true,  // auto ack
	// 	false, // exclusive
	// 	false, // no local
	// 	false, // no wait
	// 	nil,   // args
	// )
	// if err != nil {
	// 	sessn.logError("Failed to create consume " + sessn.queueName + ": " + err.Error())
	// 	sessn.isReady = false
	// 	return
	// }

	// for {
	// 	if sessn.closeMe {
	// 		return
	// 	}
	// 	select {
	// 	case msg := <-delivery:
	// 		sessn.queueDelivery <- msg
	// 	}
	// }
}

// Recv 接收消息
func (sessn *Session) Recv() (<-chan amqp.Delivery, error) {
	if !sessn.isReady {
		return nil, fmt.Errorf("no connected")
	}
	// d, ok, err := sessn.channel.Get(sessn.queueName, true)
	// if err != nil {
	// 	return nil, err
	// }
	// if !ok {
	// 	return nil, fmt.Errorf("no message")
	// }
	// return &d, nil
	return sessn.channel.Consume(
		sessn.queueName,
		"",    // Consumer
		true,  // Auto-Ack
		false, // Exclusive
		false, // No-local
		false, // No-Wait
		nil,   // Args
	)
}

// BindKey 绑定过滤器
func (sessn *Session) BindKey(k string) error {
	if sessn.isReady {
		return sessn.channel.QueueBind(sessn.queueName, k, sessn.name, false, nil)
	}
	return fmt.Errorf("Failed to bind key, channel not ready")
}

// UnBindKey 解绑过滤器
func (sessn *Session) UnBindKey(k string) error {
	if sessn.isReady {
		return sessn.channel.QueueUnbind(sessn.queueName, k, sessn.name, nil)
	}
	return fmt.Errorf("Failed to Unbind key, channel not ready")
}
func (sessn *Session) initProducer() {

}

// Send 发送数据,默认数据有效期10分钟
func (sessn *Session) Send(f string, d []byte) {
	sessn.SendCustom(&RabbitMQData{
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
func (sessn *Session) SendCustom(d *RabbitMQData) {
	if !sessn.isReady {
		return
	}
	go func() {
		defer func() { recover() }()
		err := sessn.channel.Publish(
			sessn.name,   // exchange
			d.RoutingKey, // routing key
			false,        // mandatory
			false,        // immediate
			*d.Data,
			// amqp.Publishing{
			// 	ContentType:  "text/plain",
			// 	DeliveryMode: amqp.Persistent,
			// 	Expiration:   "300000",
			// 	Timestamp:    time.Now(),
			// 	Body:         []byte(msg[1]),
			// }
		)
		if err != nil {
			sessn.writeLog("Failed to send to "+sessn.addr+": "+err.Error(), 40)
		} else {
			if sessn.loggerLevel <= 10 {
				sessn.writeLog("S:"+sessn.addr+"|"+d.RoutingKey, 10)
			}
		}
	}()
}
