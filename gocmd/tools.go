package gocmd

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/xyzj/gopsu/pathtool"
)

type ProcessInfo struct {
	Pid     int
	Name    string
	CmdLine string
}

// ProcessExist only for linux
func ProcessExist(pid int) bool {
	return pathtool.IsExist(fmt.Sprintf("/proc/%d", pid))
}

// QueryProcess only for linux
func QueryProcess(name string) []*ProcessInfo {
	pi := make([]*ProcessInfo, 0)
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return pi
	}
	for _, proc := range procs {
		if !proc.IsDir() {
			continue
		}
		b, err := os.ReadFile("/proc/" + proc.Name() + "/comm")
		if err != nil {
			continue
		}
		if name != string(bytes.TrimSpace(b)) {
			continue
		}
		pid, _ := strconv.ParseInt(proc.Name(), 10, 32)
		if pid == 0 {
			continue
		}
		cmd, _ := os.ReadFile("/proc/" + proc.Name() + "/cmdline")
		if len(cmd) == 0 {
			continue
		}
		pi = append(pi, &ProcessInfo{
			Name:    name,
			Pid:     int(pid),
			CmdLine: string(bytes.ReplaceAll(cmd, []byte{0}, []byte{32})),
		})
	}
	return pi
}
