package gocmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xyzj/gopsu/pathtool"
)

type SignalQuit struct {
	sigc   chan os.Signal
	ctx    context.Context
	cancel context.CancelFunc
}

func NewSignalQuit() *SignalQuit {
	ctx, cancel := context.WithCancel(context.Background())
	return &SignalQuit{
		sigc:   make(chan os.Signal, 1),
		ctx:    ctx,
		cancel: cancel,
	}
}

func logMessage(s string) string {
	return fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05.000"), s)
}

// SignalCapture 创建一个退出信号捕捉器
func (s *SignalQuit) SignalCapture(onSignalQuit func()) {
	signal.Notify(s.sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGHUP)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				println(fmt.Sprintf("%+v", err.(error)))
				s.cancel()
				os.Exit(1)
			}
			s.cancel()
			os.Exit(0)
		}()
		sig := <-s.sigc // 监听关闭
		w := os.Stdout
		w.WriteString("\n")
		var err error
		if os.Getenv("GOCMD_LOG_SIGNAL_TO_FILE") == "1" {
			w, err = os.OpenFile(pathtool.JoinPathFromHere(pathtool.GetExecName()+".signal.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o664)
			if err != nil {
				println(err.Error())
				w = os.Stdout
			}
		}
		w.WriteString(logMessage("got the signal `" + sig.String() + "`, shutting down.\n"))
		if onSignalQuit != nil {
			w.WriteString(logMessage("doing clean up jobs\n"))
			onSignalQuit()
		}
	}()
}

// SendSignalQuit 发送关闭信号
func (s *SignalQuit) SendSignalQuit() {
	println("\nthe program ask to quit")
	s.sigc <- syscall.SIGQUIT
	select {
	case <-time.After(time.Second * 3):
		println("waiting for quit timeout")
		os.Exit(1)
	case <-s.ctx.Done():
	}
}
