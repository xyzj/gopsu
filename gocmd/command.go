package gocmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"

	"github.com/xyzj/gopsu/pathtool"
)

// Command a command struct
type Command struct {
	// RunWithExitCode When the exitcode != -1, the framework will call the os.Exit(exitcode) method to exit the program
	RunWithExitCode func(*ProcInfo) int
	// Name command name, something like run, start, stop
	Name string
	// Descript command description, show in help message
	Descript string
	// HelpMsg set the command help message
	HelpMsg string
}

func (c *Command) printHelp() {
	if c.HelpMsg == "" {
		c.HelpMsg = fmt.Sprintf(`Usage:
    %s %s [flags]`, exeName, c.Name)
	}
	println(c.Descript, "\n\n", c.HelpMsg)
	// if flag.CommandLine.Parsed() {
	println("\nFlags:")
	flag.CommandLine.PrintDefaults()
	// }
}

var (
	exeName = pathtool.GetExecNameWithoutExt()
	// CmdStart default start command，start the program in the background
	CmdStart = &Command{
		Name:     "start",
		Descript: "start the program in the background.",
		RunWithExitCode: func(pinfo *ProcInfo) int {
			return start(pinfo)
		},
	}

	// CmdStop default stop command
	CmdStop = &Command{
		Name:     "stop",
		Descript: "stop the program with the given pid.",
		HelpMsg:  fmt.Sprintf("Usage:\n\t%s stop", exeName),
		RunWithExitCode: func(pinfo *ProcInfo) int {
			return stop(pinfo)
		},
	}

	// CmdRestart default restart command
	CmdRestart = &Command{
		Name:     "restart",
		Descript: "stop the program with the given pid, then start the program in the background.",
		RunWithExitCode: func(pinfo *ProcInfo) int {
			stop(pinfo)
			return start(pinfo)
		},
	}

	// CmdRun default run command
	CmdRun = &Command{
		Name:     "run",
		Descript: "run the program.",
		RunWithExitCode: func(pinfo *ProcInfo) int {
			pinfo.Pid = os.Getpid()
			pinfo.Save()
			SignalCapture(pinfo.Pfile, false, pinfo.onSignalQuit)
			return -1
		},
	}

	CmdStatus = &Command{
		Name:     "status",
		Descript: "chek process status",
		HelpMsg:  fmt.Sprintf("Usage:\n\t%s status", exeName),
		RunWithExitCode: func(pinfo *ProcInfo) int {
			return status(pinfo)
		},
	}
)

func start(pinfo *ProcInfo) int {
	if id, _ := pinfo.Load(false); id > 1 {
		if pp, err := os.FindProcess(id); err == nil {
			err = pp.Signal(syscall.Signal(0))
			if err == nil {
				println(fmt.Sprintf("%s already start with pid: %d", pinfo.name, id))
				return 1
			}
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

func stop(pinfo *ProcInfo) int {
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
	os.Remove(pinfo.Pfile)
	pinfo.onSignalQuit()
	return 0
}

func status(pinfo *ProcInfo) int {
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
