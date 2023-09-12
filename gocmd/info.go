package gocmd

import (
	"io"
	"os"
	"strings"

	"github.com/xyzj/gopsu/json"
)

// Info program information
type Info struct {
	// LogWriter 日志写入器
	LogWriter io.Writer
	// Ver program version
	Ver string
	// Title program title
	Title string
	// Descript program descript message
	Descript string
}

type procInfo struct {
	// os.Args[1:]
	params []string `json:"-"`
	// Args "run"+os.Args[2:]
	Args []string `json:"args"`
	// fullpath of the program
	exec string `json:"-"`
	// filepath.Base(exec)
	name string `json:"-"`
	// filepath.Dir(exec)
	dir string `json:"-"`
	// the file save the pid
	pfile string `json:"-"`
	// Pid value
	Pid          int `json:"pid"`
	onSignalQuit func()
}

// Save 保存pid信息
func (p *procInfo) Save() {
	b, _ := json.Marshal(p)
	os.WriteFile(p.pfile, b, 0664)
}

// Load 读取pid信息和启动参数
func (p *procInfo) Load(printErr bool) (int, error) {
	b, err := os.ReadFile(p.pfile)
	if err != nil {
		if printErr {
			println("failed to load pid file", err.Error())
		}
		return -1, err
	}
	pi := &procInfo{}
	err = json.Unmarshal(b, pi)
	if err != nil {
		if printErr {
			println("failed to parse pid data", err.Error())
		}
		return -1, err
	}
	if len(p.Args) == 0 {
		p.Args = pi.Args
	}
	p.Pid = pi.Pid
	return p.Pid, nil
}

// VersionInfo show something
//
// name: program name
// ver: program version
// gover: golang version
// buildDate: build datetime
// buildOS: platform info
// auth: auth name
type VersionInfo struct {
	Name         string   `json:"name,omitempty"`
	Version      string   `json:"version,omitempty"`
	GoVersion    string   `json:"go_version,omitempty"`
	BuildDate    string   `json:"build_date,omitempty"`
	BuildOS      string   `json:"build_os,omitempty"`
	CodeBy       string   `json:"code_by,omitempty"`
	StartWith    string   `json:"start_with,omitempty"`
	Dependencies []string `json:"deps,omitempty"`
}

// PrintVersion show something
//
// name: program name
// ver: program version
// gover: golang version
// buildDate: build datetime
// buildOS: platform info
// auth: auth name
func PrintVersion(v *VersionInfo) string {
	if v.StartWith == "" {
		v.StartWith = strings.Join(os.Args[1:], " ")
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return json.ToString(b)
}
