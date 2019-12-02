package main

import (
	"github.com/xyzj/gopsu/test/lib"
	"gitlab.local/wlstmicro"
)

// 启动文件 main.go
func main() {
	// 启动生产者，用于发送消息
	wlstmicro.StartMQConsumer()

	// 启动消费者，用于接收消息
	lib.NewMQConsumer()
}
