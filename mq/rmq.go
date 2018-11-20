package mq

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/xyzj/mxgo"
)

// RabbitMQ rabbit-mq struct
type RabbitMQ struct {
	Log               *mxgo.MxLog         // 日志
	Verbose           bool                // 是否打印信息
	chanSend          chan *RabbitMQData  // 发送队列
	chanRecv          chan *amqp.Delivery // 接收队列
	chanWatcher       chan string         // 子线程监视通道
	Producer          *RabbitMQArgs       // 生产者
	Consumer          *RabbitMQArgs       // 消费者
	chanCloseProducer chan bool
	chanCloseConsumer chan bool
}

// RabbitMQArgs rabbit-mq connect args
type RabbitMQArgs struct {
	ConnStr      string   // 连接字符串
	ExchangeName string   // 交换机名称
	ExchangeType string   // 交换机类型
	RoutingKeys  []string // 过滤器
	QueueName    string   // 队列名
	QueueDurable bool     // 队列是否持久化
	QueueMax     int32    // 队列长度
	ChannelCache int      // 通道大小，默认2k
}

// RabbitMQData rabbit-mq data send struct
type RabbitMQData struct {
	RoutingKey string
	Data       amqp.Publishing
}

// CloseAll close all
func (r *RabbitMQ) CloseAll() {
	r.chanCloseProducer <- true
	r.chanCloseConsumer <- true
}

func (r *RabbitMQ) coreWatcher() {
	defer func() {
		if err := recover(); err != nil {
			ioutil.WriteFile(fmt.Sprintf("crash-rmq-%s.log", time.Now().Format("20060102150405")), []byte(fmt.Sprintf("%v", errors.WithStack(err.(error)))), 0644)
			time.Sleep(300 * time.Millisecond)
		}
	}()
	var closehandle = make(map[string]bool)
	var closeme = false
	for {
		for _, v := range closehandle {
			if v == false {
				closeme = false
				break
			}
		}
		if closeme == true {
			break
		}
		select {
		case n := <-r.chanWatcher:
			time.Sleep(100 * time.Millisecond)
			switch n {
			case "producer":
				go r.handleProducer()
				closehandle["producer"] = false
			case "consumer":
				go r.handleConsumer()
				closehandle["consumer"] = false
			case "closeproducer":
				closehandle["producer"] = true
			case "closeconsumer":
				closehandle["consumer"] = true
			}
		}
	}
}

func (r *RabbitMQ) showMessages(s string, level int) {
	if r.Log != nil {
		switch level {
		case 10:
			r.Log.Debug(s)
		case 20:
			r.Log.Info(s)
		case 30:
			r.Log.Warning(s)
		case 40:
			r.Log.Error(s)
		case 90:
			r.Log.System(s)
		}
	}
	if r.Verbose {
		println(s)
	}
}

// Recv 接收数据
func (r *RabbitMQ) Recv() *amqp.Delivery {
	return <-r.chanRecv
}

// CloseConsumer close Consumer
func (r *RabbitMQ) CloseConsumer() {
	r.chanCloseConsumer <- true
}

// StartConsumer 启动消费者线程
func (r *RabbitMQ) StartConsumer() {
	if r.chanWatcher == nil {
		r.chanWatcher = make(chan string, 2)
		go r.coreWatcher()
	}
	if r.Consumer.ChannelCache == 0 {
		r.Consumer.ChannelCache = 2000
	}
	r.chanRecv = make(chan *amqp.Delivery, r.Consumer.ChannelCache)
	r.chanCloseConsumer = make(chan bool, 2)
	go r.handleConsumer()
}

// 启动消费者线程
func (r *RabbitMQ) handleConsumer() {
	defer func() {
		if err := recover(); err != nil {
			r.showMessages(fmt.Sprintf("RMQ Consumer goroutine crash: %s", err.(error).Error()), 40)
			r.chanWatcher <- "consumer"
		} else {
			r.chanWatcher <- "closeconsumer"
		}
	}()
	conn, err := amqp.Dial(r.Consumer.ConnStr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		r.Consumer.ExchangeName, // name
		r.Consumer.ExchangeType, // type
		true,                    // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		panic(err)
	}
	q, err := ch.QueueDeclare(
		r.Consumer.QueueName,    // name
		r.Consumer.QueueDurable, // durable
		false,                   // delete when usused
		false,                   // exclusive
		false,                   // no-wait
		amqp.Table{
			"x-max-length": r.Consumer.QueueMax,
		}, // arguments
	)
	if err != nil {
		ch.QueueDelete("tcs_recv", false, false, true)
		panic(err)
	}
	for _, v := range r.Consumer.RoutingKeys {
		err = ch.QueueBind(q.Name, // queue name
			v,                       // routing key
			r.Consumer.ExchangeName, // exchange
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
	r.showMessages(fmt.Sprintf("%s RMQ Consumer connect to Rabbit-MQ Server.", mxgo.Stamp2Time(time.Now().Unix())[:10]), 90)
	closeme := false
	for {
		if closeme {
			break
		}
		select {
		case msg := <-chanMsgs:
			r.chanRecv <- &msg
		case <-r.chanCloseConsumer:
			closeme = true
		}
	}
	// for {
	// 	for msg := range chanMsgs {
	// 		r.chanRecv <- &msg
	// 	}
	// }
}

// Send 发送数据
//
// amqp.Publishing{
// 	ContentType:  "text/plain",
// 	DeliveryMode: amqp.Persistent,
// 	Expiration:   "300000",
// 	Timestamp:    time.Now(),
// 	Body:         []byte("abcd"),
// },
func (r *RabbitMQ) Send(d *RabbitMQData) {
	if r.chanSend == nil {
		return
	}
	r.chanSend <- d
}

// SendGo 使用线程发送
func (r *RabbitMQ) SendGo(d *RabbitMQData) {
	go r.Send(d)
}

// CloseProducer close Producer
func (r *RabbitMQ) CloseProducer() {
	r.chanCloseProducer <- true
}

// StartProducer 启动生产者线程
func (r *RabbitMQ) StartProducer() {
	if r.chanWatcher == nil {
		r.chanWatcher = make(chan string, 2)
		go r.coreWatcher()
	}
	if r.Producer.ChannelCache == 0 {
		r.Producer.ChannelCache = 2000
	}
	r.chanSend = make(chan *RabbitMQData, r.Producer.ChannelCache)
	r.chanCloseProducer = make(chan bool, 2)
	go r.handleProducer()
}

// 启动生产者线程
func (r *RabbitMQ) handleProducer() {
	defer func() {
		if err := recover(); err != nil {
			r.showMessages(fmt.Sprintf("RMQ Producer goroutine crash: %s", err.(error).Error()), 40)
			r.chanWatcher <- "producer"
		} else {
			r.chanWatcher <- "closeproducer"
		}
	}()
	conn, err := amqp.Dial(r.Producer.ConnStr)
	if err != nil {
		panic(err)
		// tcsLog.Error(fmt.Sprintf("Failed to connect to RabbitMQ: %s", err.Error()))
		// return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		r.Producer.ExchangeName, // name
		r.Producer.ExchangeType, // type
		true,                    // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		panic(err)
	}

	r.showMessages(fmt.Sprintf("%s RMQ Producer connect to Rabbit-MQ Server.", mxgo.Stamp2Time(time.Now().Unix())[:10]), 90)
	closeme := false
	for {
		if closeme {
			break
		}
		select {
		case msg := <-r.chanSend:
			err = ch.Publish(
				r.Producer.ExchangeName, // exchange
				msg.RoutingKey,          // routing key
				false,                   // mandatory
				false,                   // immediate
				msg.Data,
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
		case <-r.chanCloseProducer:
			closeme = true
		}
	}
}
