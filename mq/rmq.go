package mq

import (
	"fmt"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"github.com/xyzj/mxgo"
)

type RabbitMQ struct {
	Log                   *mxgo.MxLog         // 日志
	Quiet                 bool                // 是否打印信息
	chanRMQSend           chan *RabbitMQData  //  发送队列
	chanRMQRecv           chan *amqp.Delivery // 接收队列
	lockRMQProducerHandle sync.WaitGroup      // rmq生产者线程监视锁
	lockRMQConsumerHandle sync.WaitGroup      // rmq消费者线程监视锁
	Producer              *RabbitMQArgs       // 生产者
	Consumer              *RabbitMQArgs       // 消费者
}

type RabbitMQArgs struct {
	ConnStr      string   // 连接字符串
	ExchangeName string   // 交换机名称
	ExchangeType string   // 交换机类型
	RoutingKeys  []string // 过滤器
	QueueName    string   // 队列名
	QueueDurable bool     // 队列是否持久化
	QueueMax     int32    // 队列长度
}

type RabbitMQData struct {
	RoutingKey string
	Data       amqp.Publishing
}

func (r *RabbitMQ) SetLogger(l *mxgo.MxLog, quiet bool) {
	if l != nil {
		r.Log = l
	}
	r.Quiet = quiet
}

// 接收数据
func (r *RabbitMQ) Recv() *amqp.Delivery {
	return <-r.chanRMQRecv
}

// 发送数据
//
// amqp.Publishing{
// 	ContentType:  "text/plain",
// 	DeliveryMode: amqp.Persistent,
// 	Expiration:   "300000",
// 	Timestamp:    time.Now(),
// 	Body:         []byte("abcd"),
// },
func (r *RabbitMQ) Send(d *RabbitMQData) {
	r.chanRMQSend <- d
}

// 使用线程发送
func (r *RabbitMQ) SendGo(d *RabbitMQData) {
	go r.Send(d)
}

func (r *RabbitMQ) StartConsumer() {
	r.chanRMQRecv = make(chan *amqp.Delivery, 1000)
	go r.waitConsumerHandle()
	go r.handleConsumer()
}

func (r *RabbitMQ) StartProducer() {
	r.chanRMQSend = make(chan *RabbitMQData, 1000)
	go r.waitProducerHandle()
	go r.handleProducer()
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
	if !r.Quiet {
		println(s)
	}
}

// 消费者线程状态监控
func (r *RabbitMQ) waitConsumerHandle() {
	for {
		time.Sleep(3 * time.Second)
		r.lockRMQConsumerHandle.Wait()
		time.Sleep(10 * time.Second)
		go r.handleConsumer()
	}
}

// 启动消费者线程
func (r *RabbitMQ) handleConsumer() {
	defer func() {
		if err := recover(); err != nil {
			r.showMessages(fmt.Sprintf("rmq consumer handle crash: %s", err.(error).Error()), 40)
		}
		r.lockRMQConsumerHandle.Done()
	}()
	r.lockRMQConsumerHandle.Add(1)
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
	msgs, err := ch.Consume(q.Name, // queue
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
	for {
		for msg := range msgs {
			r.chanRMQRecv <- &msg
		}
	}
}

// 生产者线程监控
func (r *RabbitMQ) waitProducerHandle() {
	for {
		time.Sleep(1 * time.Second)
		r.lockRMQProducerHandle.Wait()
		time.Sleep(10 * time.Second)
		go r.handleProducer()
	}
}

// 启动生产者线程
func (r *RabbitMQ) handleProducer() {
	defer func() {
		if err := recover(); err != nil {
			r.showMessages(fmt.Sprintf("RMQ Producer handle crash: %s", err.(error).Error()), 30)
		}
		r.lockRMQProducerHandle.Done()
	}()
	r.lockRMQProducerHandle.Add(1)
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

	for {
		select {
		case msg := <-r.chanRMQSend:
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
		}
	}
}
