package gocmd

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalCapture 创建一个退出信号捕捉器
func SignalCapture(pfile string, onSignalQuit func()) {
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func(c chan os.Signal) {
		defer func() {
			if err := recover(); err != nil {
				os.Exit(1)
			}
		}()
		sig := <-c // 监听关闭
		println("\ngot the signal " + sig.String() + ": shutting down.")
		os.Remove(pfile)
		if onSignalQuit != nil {
			onSignalQuit()
		}
		os.Exit(0)
	}(sigc)
}

// SignalQuit 发送关闭信号
func SignalQuit() {
	println("\nthe program ask to quit")
	sigc <- syscall.SIGQUIT
}
