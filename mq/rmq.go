package mq

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/xyzj/gopsu"
)

// RMQConsumer rabbit-mq Consumer struct
type RMQConsumer struct {
	logRmq      *gopsu.MxLog        // 日志
	verbose     bool                // 是否打印信息
	chanRecv    chan *amqp.Delivery // 接收队列
	chanClose   chan bool           // 子线程close
	chanWatcher chan string         // 子线程监视通道

	connStr      string   // 连接字符串
	exchangeName string   // 交换机名称
	exchangeType string   // 交换机类型
	routingKeys  []string // 过滤器
	queueName    string   // 队列名
	queueDurable bool     // 队列是否持久化
	queueDelete  bool     // 队列在不用时是否删除
	queueMax     int32    // 队列长度
	addr         string   // ip:port
}

// RMQProducer 生产者
type RMQProducer struct {
	logRmq      *gopsu.MxLog       // 日志
	verbose     bool               // 是否打印信息
	chanSend    chan *RabbitMQData // send queue
	chanClose   chan bool          // 子线程close
	chanWatcher chan string        // 子线程监视通道

	connStr      string // 连接字符串
	exchangeName string // 交换机名称
	exchangeType string // 交换机类型
	addr         string // ip:port
}

// RabbitMQData rabbit-mq data send struct
type RabbitMQData struct {
	RoutingKey string
	Data       *amqp.Publishing
}

// NewConsumer 新的消费者
func NewConsumer(conn, exchangeName, exchangeType, queueName string, routingKeys []string) *RMQConsumer {
	return &RMQConsumer{
		connStr:      conn,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		queueName:    queueName,
		routingKeys:  routingKeys,
		queueDurable: true,
		queueMax:     70000,
		chanRecv:     make(chan *amqp.Delivery, 2000),
		chanClose:    make(chan bool, 2),
		chanWatcher:  make(chan string, 2),
		addr:         strings.Split(conn, "@")[1],
	}
}

// SetLogger 设置日志
func (r *RMQConsumer) SetLogger(l *gopsu.MxLog) {
	r.logRmq = l
}

// SetDebug 设置是否调试
func (r *RMQConsumer) SetDebug(b bool) {
	r.verbose = b
}

// Recv 接收数据
func (r *RMQConsumer) Recv() *amqp.Delivery {
	return <-r.chanRecv
}

// Start Start consumer
func (r *RMQConsumer) Start() {
	go r.coreWatcher()
	r.chanWatcher <- "consumer"
}

// Close 关闭消费者
func (r *RMQConsumer) Close() {
	r.chanClose <- true
}

// String 返回ip:port
func (r *RMQConsumer) String() string {
	return r.addr
}

func (r *RMQConsumer) coreWatcher() {
	defer func() {
		if err := recover(); err != nil {
			ioutil.WriteFile(fmt.Sprintf("crash-rmq-%s.log", time.Now().Format("20060102150405")), []byte(fmt.Sprintf("%v", errors.WithStack(err.(error)))), 0644)
			time.Sleep(300 * time.Millisecond)
		}
	}()
	var closeme = false
	for {
		if closeme == true {
			break
		}
		select {
		case n := <-r.chanWatcher:
			time.Sleep(100 * time.Millisecond)
			switch n {
			case "consumer":
				for {
					conn, err := r.initConsumer()
					if err != nil {
						time.Sleep(15 * time.Second)
					} else {
						go r.handleConsumer(conn)
						break
					}
				}
				closeme = false
			case "closeconsumer":
				closeme = true
			}
		}
	}
}

func (r *RMQConsumer) showMessages(s string, level int) {
	if r.logRmq != nil {
		switch level {
		case 10:
			r.logRmq.Debug(s)
		case 20:
			r.logRmq.Info(s)
		case 30:
			r.logRmq.Warning(s)
		case 40:
			r.logRmq.Error(s)
		case 90:
			r.logRmq.System(s)
		}
	}
	if r.verbose {
		println(s)
	}
}

func (r *RMQConsumer) initConsumer() (*amqp.Connection, error) {
	conn, err := amqp.Dial(r.connStr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// 启动消费者线程
func (r *RMQConsumer) handleConsumer(conn *amqp.Connection) {
	defer func() {
		if err := recover(); err != nil {
			r.showMessages(fmt.Sprintf("RMQ Consumer goroutine crash: %s", err.(error).Error()), 40)
			r.chanWatcher <- "consumer"
		} else {
			r.chanWatcher <- "closeconsumer"
		}
	}()
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		r.exchangeName, // name
		r.exchangeType, // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		panic(err)
	}
	q, err := ch.QueueDeclare(
		r.queueName,    // name
		r.queueDurable, // durable
		r.queueDelete,  // delete when unused
		false,          // exclusive
		false,          // no-wait
		amqp.Table{
			"x-max-length": r.queueMax,
		}, // arguments
	)
	if err != nil {
		ch.QueueDelete(r.exchangeName, false, false, true)
		panic(err)
	}
	for _, v := range r.routingKeys {
		err = ch.QueueBind(q.Name, // queue name
			v,              // routing key
			r.exchangeName, // exchange
			false,
			nil)
		if err != nil {
			panic(err)
		}
	}
	chanMsgs, err := ch.Consume(q.Name, // queue
		"",    // consumer
		true,  // auto ack
		false, // exclusive
		false, // no local
		false, // no wait
		nil,   // args
	)
	if err != nil {
		panic(err)
	}
	r.showMessages(fmt.Sprintf("%s RMQ Consumer connect to Rabbit-MQ Server.", gopsu.Stamp2Time(time.Now().Unix())[:10]), 90)
	closeme := false
	for {
		if closeme {
			break
		}
		select {
		case msg := <-chanMsgs:
			r.chanRecv <- &msg
		case <-r.chanClose:
			closeme = true
		}
	}
}

// NewProducer 新的生产
func NewProducer(conn, exchangeName, exchangeType string) *RMQProducer {
	return &RMQProducer{
		connStr:      conn,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		chanSend:     make(chan *RabbitMQData, 5000),
		chanClose:    make(chan bool, 2),
		chanWatcher:  make(chan string, 2),
		addr:         strings.Split(conn, "@")[1],
	}
}

// SetLogger 设置日志
func (r *RMQProducer) SetLogger(l *gopsu.MxLog) {
	r.logRmq = l
}

// SetDebug 设置是否调试
func (r *RMQProducer) SetDebug(b bool) {
	r.verbose = b
}

// Start Start producer
func (r *RMQProducer) Start() {
	go r.coreWatcher()
	r.chanWatcher <- "producer"
}

// Close 关闭消费者
func (r *RMQProducer) Close() {
	r.chanClose <- true
}

// String 返回ip:port
func (r *RMQProducer) String() string {
	return r.addr
}

func (r *RMQProducer) coreWatcher() {
	defer func() {
		if err := recover(); err != nil {
			ioutil.WriteFile(fmt.Sprintf("crash-rmq-%s.log", time.Now().Format("20060102150405")), []byte(fmt.Sprintf("%v", errors.WithStack(err.(error)))), 0644)
			time.Sleep(300 * time.Millisecond)
		}
	}()
	var closeme = false
	for {
		if closeme == true {
			break
		}
		select {
		case n := <-r.chanWatcher:
			time.Sleep(100 * time.Millisecond)
			switch n {
			case "producer":
				for {
					conn, err := r.initProducer()
					if err != nil {
						time.Sleep(15 * time.Second)
					} else {
						go r.handleProducer(conn)
						break
					}
				}
				closeme = false
			case "closeproducer":
				closeme = true
			}
		}
	}
}

func (r *RMQProducer) showMessages(s string, level int) {
	if r.logRmq != nil {
		switch level {
		case 10:
			r.logRmq.Debug(s)
		case 20:
			r.logRmq.Info(s)
		case 30:
			r.logRmq.Warning(s)
		case 40:
			r.logRmq.Error(s)
		case 90:
			r.logRmq.System(s)
		}
	}
	if r.verbose {
		println(s)
	}
}

func (r *RMQProducer) initProducer() (*amqp.Connection, error) {
	conn, err := amqp.Dial(r.connStr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Send 发送数据,默认数据有效期10分钟
func (r *RMQProducer) Send(f string, d []byte) {
	r.SendCustom(&RabbitMQData{
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
func (r *RMQProducer) SendCustom(d *RabbitMQData) {
	if r.chanSend == nil {
		return
	}
	go func() {
		defer func() { recover() }()
		r.chanSend <- d
	}()
}

// 启动生产者线程
func (r *RMQProducer) handleProducer(conn *amqp.Connection) {
	defer func() {
		if err := recover(); err != nil {
			r.showMessages(fmt.Sprintf("RMQ Producer goroutine crash: %s", err.(error).Error()), 40)
			r.chanWatcher <- "producer"
		} else {
			r.chanWatcher <- "closeproducer"
		}
	}()
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		r.exchangeName, // name
		r.exchangeType, // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		panic(err)
	}

	r.showMessages(fmt.Sprintf("%s RMQ Producer connect to Rabbit-MQ Server.", gopsu.Stamp2Time(time.Now().Unix())[:10]), 90)
	closeme := false
	for {
		if closeme {
			break
		}
		select {
		case msg := <-r.chanSend:
			err = ch.Publish(
				r.exchangeName, // exchange
				msg.RoutingKey, // routing key
				false,          // mandatory
				false,          // immediate
				*msg.Data,
				// amqp.Publishing{
				// 	ContentType:  "text/plain",
				// 	DeliveryMode: amqp.Persistent,
				// 	Expiration:   "300000",
				// 	Timestamp:    time.Now(),
				// 	Body:         []byte(msg[1]),
				// }
			)
			if err != nil {
				panic(err)
			}
		case <-r.chanClose:
			closeme = true
		}
	}
}
