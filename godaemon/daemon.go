package godaemon

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	god   = flag.Bool("b", false, "run program in the background")
	pname = flag.String("p", "", "save the pid file, only work with -b")
)
var (
	sigc = make(chan os.Signal, 1)
)

// Start 后台运行
//
// fQuit: 捕获信号时执行的清理工作
func Start(fQuit func()) {
	if !flag.Parsed() {
		flag.Parse()
	}
	CaughtSignal(fQuit)
	if !*god {
		return
	}
	RunBackground()
}

// RunBackground 后台运行
func RunBackground() {
	xss := make([]string, 0)
	idx := 0
	for k, v := range os.Args[1:] {
		if v == "-b" || v == "-b=true" {
			idx = k + 1
			continue
		}
		if idx > 0 && idx == k && v[0] != 45 {
			continue
		}
		xss = append(xss, v)
	}
	cmd := exec.Command(os.Args[0], xss...)
	if err := cmd.Start(); err != nil {
		println("start " + os.Args[0] + " failed, error: " + err.Error())
		os.Exit(1)
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	if *pname != "" {
		ioutil.WriteFile(*pname, []byte(pid), 0664)
	}
	println(os.Args[0] + " [PID] " + pid + " running ...")
	os.Exit(0)
}

// CaughtSignal 捕获退出信号
//
// fQuit: 捕获信号时执行的清理工作
func CaughtSignal(fQuit func()) {
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func(c chan os.Signal) {
		sig := <-c // 监听关闭
		println("got the signal " + sig.String() + ": shutting down.")
		if *pname != "" {
			os.Remove(*pname)
		}
		if fQuit != nil {
			fQuit()
		}
		time.Sleep(time.Millisecond * 777)
		os.Exit(0)
	}(sigc)
}

// SignalQuit 关闭
func SignalQuit() {
	println("the program ask to quit")
	sigc <- syscall.SIGQUIT
}
