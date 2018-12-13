package zmq

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pebbe/zmq4"
	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
)

var (
	zmqRShwm = 7000 // zmq缓存队列大小
)

// ZeroMQ zeromq
type ZeroMQ struct {
	Log           *gopsu.MxLog // 日志
	Verbose       bool        // 是否打印信息
	Pull          *ZeroMQArgs
	Push          *ZeroMQArgs
	Pub           *ZeroMQArgs
	Sub           *ZeroMQArgs
	chanPush      chan *ZeroMQData
	chanSub       chan *ZeroMQData
	chanWatcher   chan string
	chanClosePush chan bool
	chanCloseSub  chan bool
}

// ZeroMQArgs zmq args
type ZeroMQArgs struct {
	ConnStr          string        // 连接字符串
	Timeo            time.Duration // IO超时
	ChannelCache     int           // 信号通道大小，默认2k
	Subscribe        []string      //  sub过滤器
	ReconnectIfTimeo bool
}

// ZeroMQData zmq data
type ZeroMQData struct {
	RoutingKey string
	Body       []byte
}

func (z *ZeroMQ) showMessages(s string, level int) {
	if z.Log != nil {
		switch level {
		case 10:
			z.Log.Debug(s)
		case 20:
			z.Log.Info(s)
		case 30:
			z.Log.Warning(s)
		case 40:
			z.Log.Error(s)
		case 90:
			z.Log.System(s)
		}
	}
	if z.Verbose {
		println(s)
	}
}

func (z *ZeroMQ) coreWatcher() {
	defer func() {
		if err := recover(); err != nil {
			ioutil.WriteFile(fmt.Sprintf("crash-0mq-%s.log", time.Now().Format("20060102150405")), []byte(fmt.Sprintf("%v", errors.WithStack(err.(error)))), 0644)
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
		case n := <-z.chanWatcher:
			time.Sleep(100 * time.Millisecond)
			switch n {
			case "push":
				go z.handlePush()
				closehandle["push"] = false
			case "sub":
				go z.handleSub()
				closehandle["sub"] = false
			case "closepush":
				closehandle["push"] = true
			case "closesub":
				closehandle["sub"] = true
			}
		}
	}
}

// PushData push data
func (z *ZeroMQ) PushData(f string, d []byte) {
	if z.chanPush == nil {
		return
	}
	go func() {
		z.chanPush <- &ZeroMQData{
			RoutingKey: f,
			Body:       d,
		}
	}()
}

// ClosePush close push goroutine
func (z *ZeroMQ) ClosePush() {
	z.chanClosePush <- true
}

// StartPush start zmq push
func (z *ZeroMQ) StartPush() {
	if z.chanWatcher == nil {
		z.chanWatcher = make(chan string, 2)
		go z.coreWatcher()
	}
	if z.Push.ChannelCache == 0 {
		z.Push.ChannelCache = 2000
	}
	if z.Push.Timeo == 0 {
		z.Push.Timeo = 50 * time.Millisecond
	}

	z.chanPush = make(chan *ZeroMQData, z.Push.ChannelCache)
	go z.handlePush()
}
func (z *ZeroMQ) handlePush() {
	defer func() {
		if err := recover(); err != nil {
			z.showMessages(fmt.Sprintf("0MQ-Push goroutine crash: %s", err.(error).Error()), 40)
			z.chanWatcher <- "push"
		} else {
			z.chanWatcher <- "closepush"
		}
	}()
	push, _ := zmq4.NewSocket(zmq4.PUSH)
	defer push.Close()
	push.SetSndhwm(zmqRShwm)
	push.SetSndtimeo(z.Push.Timeo)
	// push.SetLinger(0)
	push.Connect(z.Push.ConnStr)
	z.showMessages(fmt.Sprintf("%s 0MQ-Push connect to %s", gopsu.Stamp2Time(time.Now().Unix(), "2006-01-02"), z.Push.ConnStr), 90)

	closeme := false
	for {
		if closeme {
			break
		}
		select {
		case msg := <-z.chanPush:
			_, ex := push.SendMessage(msg)
			if ex != nil {
				z.showMessages(fmt.Sprintf("0MQ-PushEx:%s", ex.Error()), 40)
			}
		case msg := <-z.chanClosePush:
			if msg {
				closeme = true
			}
		}
	}
}

// SubData sub data use channel
func (z *ZeroMQ) SubData() *ZeroMQData {
	return <-z.chanSub
}

// StartSub start zmq sub
func (z *ZeroMQ) StartSub() {
	if z.chanWatcher == nil {
		z.chanWatcher = make(chan string, 2)
		go z.coreWatcher()
	}
	if z.Sub.ChannelCache == 0 {
		z.Sub.ChannelCache = 2000
	}
	if z.Sub.Timeo <= 0 {
		z.Sub.Timeo = 5 * time.Second
	}

	z.chanCloseSub = make(chan bool)
	z.chanSub = make(chan *ZeroMQData, z.Sub.ChannelCache)
	go z.handleSub()
}
func (z *ZeroMQ) handleSub() {
	defer func() {
		if err := recover(); err != nil {
			z.showMessages(fmt.Sprintf("0MQ-Sub goroutine crash: %s", err.(error).Error()), 40)
			z.chanWatcher <- "sub"
		} else {
			z.chanWatcher <- "closesub"
		}
	}()
	sub, _ := zmq4.NewSocket(zmq4.SUB)
	sub.SetRcvhwm(zmqRShwm)
	sub.SetLinger(0)
	sub.SetRcvtimeo(z.Sub.Timeo)
	if len(z.Sub.Subscribe) == 0 {
		sub.SetSubscribe("")
	} else {
		for _, v := range z.Sub.Subscribe {
			sub.SetSubscribe(v)
		}
	}
	sub.Connect(z.Sub.ConnStr)
	z.showMessages(fmt.Sprintf("%s 0MQ-Sub connect to %s", gopsu.Stamp2Time(time.Now().Unix(), "2006-01-02"), z.Sub.ConnStr), 90)
	closeme := false
	go func() {
		closeme = <-z.chanCloseSub
	}()
	for {
		if closeme {
			break
		}
		msg, ex := sub.RecvMessageBytes(0)
		if ex != nil {
			if z.Sub.ReconnectIfTimeo {
				sub.Close()
				z.showMessages("ZMQ-SUB recv timeout, try reconnect", 40)
			}
			continue
		}
		if len(msg) > 1 {
			z.chanSub <- &ZeroMQData{
				RoutingKey: string(msg[0]),
				Body:       msg[1],
			}
		}
	}
}

// StartProxy start a zmq proxy
func (z *ZeroMQ) StartProxy() {
	frontend, _ := zmq4.NewSocket(zmq4.PULL)
	frontend.SetReconnectIvl(70 * time.Second)
	frontend.SetRcvhwm(zmqRShwm)
	frontend.SetLinger(0)
	defer frontend.Close()
	if err := frontend.Bind(z.Pull.ConnStr); err != nil {
		z.showMessages(fmt.Sprintf("0MQ-Binding %s failed: %+v", z.Pull.ConnStr, err), 40)
		return
	}

	//  Socket facing services
	backend, _ := zmq4.NewSocket(zmq4.PUB)
	backend.SetReconnectIvl(70 * time.Second)
	backend.SetSndhwm(zmqRShwm)
	backend.SetLinger(0)
	defer backend.Close()
	if err := backend.Bind(z.Pub.ConnStr); err != nil {
		z.showMessages(fmt.Sprintf("0MQ-Binding %s failed: %+v", z.Pub.ConnStr, err), 40)
		return
	}
	z.showMessages(fmt.Sprintf("%s 0MQ-Proxy start success on %s to %s", gopsu.Stamp2Time(time.Now().Unix(), "2006-01-02"), z.Pull.ConnStr, z.Pub.ConnStr), 90)
	//  Start the proxy
	if err := zmq4.Proxy(frontend, backend, nil); err != nil {
		z.showMessages(fmt.Sprintf("0MQ-Proxy interrupted: %+v", err), 40)
	}
}
