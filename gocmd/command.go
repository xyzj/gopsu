package gocmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

var (
	sigc = make(chan os.Signal, 1)
)

// Command a command struct
type Command struct {
	// RunWithExitCode When the exitcode != -1, the framework will call the os.Exit(exitcode) method to exit the program
	RunWithExitCode func(pinfo *procInfo) int
	// Name command name, something like run, start, stop
	Name string
	// Descript command description, show in help message
	Descript string
	// pinfo program running infomation
	pinfo *procInfo
}

var (
	// CmdStart default start command，start the program in the background
	CmdStart = &Command{
		Name:     "start",
		Descript: "start the program in the background.",
		RunWithExitCode: func(pinfo *procInfo) int {
			return start(pinfo)
		},
	}

	// CmdStop default stop command
	CmdStop = &Command{
		Name:     "stop",
		Descript: "stop the program with the given pid.",
		RunWithExitCode: func(pinfo *procInfo) int {
			return stop(pinfo)
		},
	}

	// CmdRestart default restart command
	CmdRestart = &Command{
		Name:     "restart",
		Descript: "stop the program with the given pid, then start the program in the background.",
		RunWithExitCode: func(pinfo *procInfo) int {
			stop(pinfo)
			return start(pinfo)
		},
	}

	// CmdRun default run command
	CmdRun = &Command{
		Name:     "run",
		Descript: "run the program.",
		RunWithExitCode: func(pinfo *procInfo) int {
			caughtSignal(pinfo)
			return -1
		},
	}
)

func start(pinfo *procInfo) int {
	if id, _ := readPID(pinfo.pfile, false); id > 1 {
		if _, err := os.FindProcess(id); err == nil {
			println(fmt.Sprintf("%s already start with pid: %d", pinfo.name, id))
			return 1
		}
	}
	pinfo.args = []string{"run"}
	pinfo.args = append(pinfo.args, pinfo.params[1:]...)
	cmd := exec.Command(pinfo.exec, pinfo.args...)
	cmd.Dir = pinfo.dir
	if err := cmd.Start(); err != nil {
		println("start " + pinfo.name + " failed, error: " + err.Error())
		return 1
	}
	pinfo.pid = strconv.Itoa(cmd.Process.Pid)
	os.WriteFile(pinfo.pfile, []byte(pinfo.pid), 0664)
	println(pinfo.name + " [PID] " + pinfo.pid + " running ...")
	return 0
}

func stop(pinfo *procInfo) int {
	id, err := readPID(pinfo.pfile, true)
	if err != nil {
		return 1
	}
	process, err := os.FindProcess(id)
	if err != nil {
		println("failed to find process by pid: %d, reason: %v", id, process)
		return 1
	}
	err = process.Kill()
	if err != nil {
		println("failed to kill process %d: %v", id, err)
	} else {
		println("killed process: ", id)
	}
	os.Remove(pinfo.pfile)
	return 0
}

func caughtSignal(pinfo *procInfo) {
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func(c chan os.Signal) {
		sig := <-c // 监听关闭
		println("got the signal " + sig.String() + ": shutting down.")
		os.Remove(pinfo.pfile)
		os.Exit(0)
	}(sigc)
}
