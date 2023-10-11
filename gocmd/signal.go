package gocmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xyzj/gopsu/pathtool"
)

var (
	sigc = make(chan os.Signal, 1)
)

func logMessage(s string) []byte {
	return []byte(fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05.000"), s))
}

// SignalCapture 创建一个退出信号捕捉器
func SignalCapture(pfile string, logSignalToFile bool, onSignalQuit func()) {
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT)
	go func(c chan os.Signal) {
		sig := <-c // 监听关闭
		w := os.Stdout
		var err error
		if logSignalToFile {
			w, err = os.OpenFile(pathtool.JoinPathFromHere(pathtool.GetExecName()+".signal.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0664)
			if err != nil {
				println(err.Error())
				w = os.Stdout
			}
		}
		defer func() {
			if err := recover(); err != nil {
				w.Write(logMessage(fmt.Sprintf("%+v", err.(error))))
				os.Exit(1)
			}
		}()
		os.Remove(pfile)
		w.Write(logMessage("got the signal: " + sig.String() + ", shutting down.\n"))
		if onSignalQuit != nil {
			onSignalQuit()
		}
		os.Exit(0)
	}(sigc)
}

// SendSignalQuit 发送关闭信号
func SendSignalQuit() {
	println("\nthe program ask to quit")
	sigc <- syscall.SIGQUIT
}
