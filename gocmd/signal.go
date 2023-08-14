package gocmd

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalCapture 创建一个退出信号捕捉器
func SignalCapture(pfile string) {
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func(c chan os.Signal) {
		sig := <-c // 监听关闭
		println("\ngot the signal " + sig.String() + ": shutting down.")
		os.Remove(pfile)
		os.Exit(0)
	}(sigc)
}

// SignalQuit 发送关闭信号
func SignalQuit() {
	println("\nthe program ask to quit")
	sigc <- syscall.SIGQUIT
}
