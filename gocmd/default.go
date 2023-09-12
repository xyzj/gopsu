package gocmd

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/mapfx"
	"github.com/xyzj/gopsu/pathtool"
)

// DefaultProgram Create a default console program, this program contains commands: `start`, `stop`, `restart`, `run`, `status`, `version`, `help`
func DefaultProgram(info *Info) *Program {
	return NewProgram(info).AddCommand(CmdRun).AddCommand(CmdStart).AddCommand(CmdStop).AddCommand(CmdRestart).AddCommand(CmdStatus)
}

// NewProgram Create a empty console program, this program contains commands: `version`, `help`
func NewProgram(info *Info) *Program {
	if info == nil {
		info = &Info{
			Title:    "A general program startup framework",
			Descript: "can run program in the background",
			Ver:      "0.0.1",
		}
	}
	if info.LogWriter == nil {
		info.LogWriter = logger.NewWriter(&logger.OptLog{})
	}
	params := os.Args
	pinfo := &procInfo{
		params: make([]string, 0),
	}
	// 获取程序信息
	pinfo.params = params[1:]
	if exec, err := filepath.Abs(params[0]); err == nil {
		pinfo.exec = exec
	} else {
		pinfo.exec = params[0]
	}
	pinfo.name = pathtool.GetExecName()
	pinfo.dir = pathtool.GetExecDir()
	if info.Title == "" {
		info.Title = pinfo.name
	}
	// 处理参数
	flag.StringVar(&pinfo.pfile, "p", "", "set the pid file path")
	for k, v := range params {
		if strings.HasPrefix(v, "-") {
			flag.CommandLine.Parse(params[k:])
			pinfo.Args = params[k:]
			break
		}
	}
	// 设置pid文件
	if pinfo.pfile == "" {
		pinfo.pfile = pathtool.JoinPathFromHere(pinfo.name + ".pid")
	}
	return &Program{
		info:  info,
		cmds:  mapfx.NewUniqueSlice[*Command](),
		pinfo: pinfo,
	}
}
