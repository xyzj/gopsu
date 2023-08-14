package gocmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

var (
	sigc = make(chan os.Signal, 1)
)

// Command a command struct
type Command struct {
	// RunWithExitCode When the exitcode != -1, the framework will call the os.Exit(exitcode) method to exit the program
	RunWithExitCode func(*procInfo) int
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
			SignalCapture(pinfo.pfile)
			return -1
		},
	}

	CmdStatus = &Command{
		Name:     "status",
		Descript: "chek process status",
		RunWithExitCode: func(pinfo *procInfo) int {
			return status(pinfo)
		},
	}
)

func start(pinfo *procInfo) int {
	if id, _ := pinfo.Load(false); id > 1 {
		if _, err := os.FindProcess(id); err == nil {
			println(fmt.Sprintf("%s already start with pid: %d", pinfo.name, id))
			return 1
		}
	}
	xargs := []string{"run"}
	if len(pinfo.Args) > 0 {
		xargs = append(xargs, pinfo.Args...)
	}
	cmd := exec.Command(pinfo.exec, xargs...)
	cmd.Dir = pinfo.dir
	if err := cmd.Start(); err != nil {
		println("start " + pinfo.name + " failed, error: " + err.Error())
		return 1
	}
	pinfo.Pid = cmd.Process.Pid
	pinfo.Save()
	println(pinfo.name + " [PID] " + strconv.Itoa(pinfo.Pid) + " running ...")
	return 0
}

func stop(pinfo *procInfo) int {
	id, err := pinfo.Load(true)
	if err != nil {
		return 1
	}
	process, err := os.FindProcess(id)
	if err != nil {
		println(fmt.Sprintf("failed to find process by pid: %d, reason: %v", id, process))
		return 1
	}
	err = process.Kill()
	if err != nil {
		println(fmt.Sprintf("failed to kill process %d: %v", id, err))
	} else {
		println(fmt.Sprintf("killed process: %d", id))
	}
	os.Remove(pinfo.pfile)
	return 0
}

func status(pinfo *procInfo) int {
	id, err := pinfo.Load(true)
	if err != nil {
		return 1
	}
	_, err = os.FindProcess(id)
	if err != nil {
		println(fmt.Sprintf("failed to find process by pid: %d, reason: %v", id, err.Error()))
		return 1
	}
	if runtime.GOOS == "windows" {
		println("process " + pinfo.name + " is running by pid " + strconv.Itoa(id))
		return 0
	}
	s := []string{"-p", strconv.Itoa(id), "-o", "user=", "-o", "pid=", "-o", `%cpu=`, "-o", `%mem=`, "-o", "stat=", "-o", "start=", "-o", "time=", "-o", "cmd="}
	cmd := exec.Command("ps", s...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		println("process status error: " + err.Error())
		return 1
	}
	print(string(b))
	return 0
}
