package godaemon

import (
	"flag"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	god = flag.Bool("d", false, "run program as daemon")
)

// Start 后台运行
func Start() {
	if !flag.Parsed() {
		flag.Parse()
	}
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
		if v == "-d" || v == "-d=true" {
			idx = k + 1
			continue
		}
		if idx > 0 && idx == k && !strings.HasPrefix(v, "-") {
			continue
		}
		xss = append(xss, v)
	}
	cmd := exec.Command(os.Args[0], xss...)
	if err := cmd.Start(); err != nil {
		println("start " + os.Args[0] + " failed, error: " + err.Error())
		os.Exit(1)
	}
	println(os.Args[0] + " [PID] " + strconv.Itoa(cmd.Process.Pid) + " running ...")
	os.Exit(0)
}
