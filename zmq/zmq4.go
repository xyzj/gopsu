package zmq

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pebbe/zmq4"
	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
)

var (
	ZMQRShwm = 7000 // 0MQ缓存队列大小
)

// ZeroMQ zeromq
type ZeroMQ struct {
	Log           gopsu.Logger // 日志
	Verbose       bool         // 是否打印信息
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

// ZeroMQArgs 0MQ args
type ZeroMQArgs struct {
	ConnStr          string        // 连接字符串
	Timeo            time.Duration // IO超时
	ChannelCache     int           // 信号通道大小，默认2k
	Subscribe        []string      //  sub过滤器
	ReconnectIfTimeo bool
}

// ZeroMQData 0MQ data
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
			ioutil.WriteFile(fmt.Sprintf("crash-0mq-%s.log", time.Now().Format("20060102150405")), []byte(fmt.Sprintf("%v", errors.WithStack(err.(error)))), 0664)
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
				for {
					push, err := z.initPush()
					if err != nil {
						time.Sleep(15 * time.Second)
					} else {
						go z.handlePush(push)
						break
					}
				}
				closehandle["push"] = false
			case "sub":
				for {
					sub, err := z.initSub()
					if err != nil {
						time.Sleep(15 * time.Second)
					} else {
						go z.handleSub(sub)
						break
					}
				}
				closehandle["sub"] = false
			case "closepush":
				closehandle["push"] = true
			case "closesub":
				closehandle["sub"] = true
			}
		}
	}
}

func (z *ZeroMQ) initPush() (*zmq4.Socket, error) {
	push, _ := zmq4.NewSocket(zmq4.PUSH)
	push.SetSndhwm(ZMQRShwm)
	push.SetSndtimeo(z.Push.Timeo)
	// push.SetLinger(0)
	err := push.Connect(z.Push.ConnStr)
	if err != nil {
		z.showMessages(fmt.Sprintf("%s 0MQ-Push connect to %s failed: %s", gopsu.Stamp2Time(time.Now().Unix(), "2006-01-02"), z.Push.ConnStr, err.Error()), 40)
		return nil, err
	}
	z.showMessages(fmt.Sprintf("%s 0MQ-Push connect to %s", gopsu.Stamp2Time(time.Now().Unix(), "2006-01-02"), z.Push.ConnStr), 90)
	return push, nil
}

func (z *ZeroMQ) initSub() (*zmq4.Socket, error) {
	sub, _ := zmq4.NewSocket(zmq4.SUB)
	sub.SetRcvhwm(ZMQRShwm)
	sub.SetLinger(0)
	sub.SetRcvtimeo(z.Sub.Timeo)
	if len(z.Sub.Subscribe) == 0 {
		sub.SetSubscribe("")
	} else {
		for _, v := range z.Sub.Subscribe {
			sub.SetSubscribe(v)
		}
	}
	err := sub.Connect(z.Sub.ConnStr)
	if err != nil {
		z.showMessages(fmt.Sprintf("%s 0MQ-Sub connect to %s failed: %s", gopsu.Stamp2Time(time.Now().Unix(), "2006-01-02"), z.Sub.ConnStr, err.Error()), 40)
		return nil, err
	}
	z.showMessages(fmt.Sprintf("%s 0MQ-Sub connect to %s", gopsu.Stamp2Time(time.Now().Unix(), "2006-01-02"), z.Sub.ConnStr), 90)
	return sub, nil
}

// PushData push data
func (z *ZeroMQ) PushData(f string, d []byte) {
	if z.chanPush == nil {
		return
	}
	go func() {
		defer func() { recover() }()
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

// StartPush start 0MQ push
func (z *ZeroMQ) StartPush() error {
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

	push, err := z.initPush()
	if err != nil {
		return err
	}

	go z.handlePush(push)
	return nil
}
func (z *ZeroMQ) handlePush(push *zmq4.Socket) {
	defer func() {
		if err := recover(); err != nil {
			z.showMessages(fmt.Sprintf("0MQ-Push goroutine crash: %s", err.(error).Error()), 40)
			z.chanWatcher <- "push"
		} else {
			z.chanWatcher <- "closepush"
		}
	}()
	closeme := false
	for {
		if closeme {
			break
		}
		select {
		case msg := <-z.chanPush:
			_, ex := push.SendMessage([]string{msg.RoutingKey, gopsu.String(msg.Body)})
			if ex != nil {
				z.showMessages(fmt.Sprintf("0MQ-PushEx:%s", ex.Error()), 40)
			} else {
				if z.Verbose {
					z.showMessages(fmt.Sprintf("0MQ-Push:%s", fmt.Sprintf("%s|%s", msg.RoutingKey, base64.StdEncoding.EncodeToString(msg.Body))), 10)
				}
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

// StartSub start 0MQ sub
func (z *ZeroMQ) StartSub() error {
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

	z.chanCloseSub = make(chan bool, 3)
	z.chanSub = make(chan *ZeroMQData, z.Sub.ChannelCache)

	sub, err := z.initSub()
	if err != nil {
		return err
	}
	go z.handleSub(sub)
	return nil
}
func (z *ZeroMQ) handleSub(sub *zmq4.Socket) {
	defer func() {
		if err := recover(); err != nil {
			z.showMessages(fmt.Sprintf("0MQ-Sub goroutine crash: %s", err.(error).Error()), 40)
			z.chanWatcher <- "sub"
		} else {
			z.chanWatcher <- "closesub"
		}
	}()

	closeme := false
	go func() {
		closeme = <-z.chanCloseSub
	}()
	// fl, _ := z.Log.GetLogLevel()
	for {
		if closeme {
			break
		}
		msg, ex := sub.RecvMessageBytes(0)
		if ex != nil {
			if z.Sub.ReconnectIfTimeo {
				sub.Close()
				z.chanCloseSub <- true
				panic(errors.New("0MQ-Sub recv timeout, try reconnect"))
				// z.showMessages("0MQ-Sub recv timeout, try reconnect", 40)
			}
			continue
		}

		if len(msg) > 1 {
			z.chanSub <- &ZeroMQData{
				RoutingKey: gopsu.String(msg[0]),
				Body:       msg[1],
			}
			if z.Verbose {
				z.showMessages(fmt.Sprintf("0MQ-Sub: %s|%s", gopsu.String(msg[0]), base64.StdEncoding.EncodeToString(msg[1])), 10)
			}
		}
	}
}

// StartProxy start a 0MQ proxy
func (z *ZeroMQ) StartProxy() {
	frontend, _ := zmq4.NewSocket(zmq4.PULL)
	frontend.SetReconnectIvl(70 * time.Second)
	frontend.SetRcvhwm(ZMQRShwm)
	frontend.SetLinger(0)
	defer frontend.Close()
	if err := frontend.Bind(z.Pull.ConnStr); err != nil {
		z.showMessages(fmt.Sprintf("0MQ-Binding %s failed: %+v", z.Pull.ConnStr, err), 40)
		return
	}

	//  Socket facing services
	backend, _ := zmq4.NewSocket(zmq4.PUB)
	backend.SetReconnectIvl(70 * time.Second)
	backend.SetSndhwm(ZMQRShwm)
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
