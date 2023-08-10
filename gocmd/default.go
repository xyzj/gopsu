package gocmd

import (
	"flag"
	"os"

	"github.com/xyzj/gopsu/mapfx"
	"github.com/xyzj/gopsu/pathtool"
)

// DefaultProgram Create a default console program, this program contains commands: `start`, `stop`, `restart`, `run`, `version`, `help`
func DefaultProgram(info *Info) *Program {
	p := NewProgram(info)
	p.AddCommand(CmdStart)
	p.AddCommand(CmdStop)
	p.AddCommand(CmdRestart)
	p.AddCommand(CmdRun)
	return p
}

// NewProgram Create a empty console program, this program contains commands: `version`, `help`
func NewProgram(info *Info) *Program {
	if info == nil {
		info = &Info{
			Title:    "A general program startup framework",
			Descript: "can run program in the background",
			Ver:      "v0.0.1",
		}
	}
	params := os.Args
	pinfo := &procInfo{}
	// 获取程序信息
	pinfo.params = params[1:]
	pinfo.exec = params[0]
	pinfo.name = pathtool.GetExecName()
	pinfo.dir = pathtool.GetExecDir()
	if info.Title == "" {
		info.Title = pinfo.name
	}
	// 处理参数
	flag.StringVar(&pinfo.pfile, "p", "", "set the pid file path")
	if len(params) > 2 {
		flag.CommandLine.Parse(params[2:])
	}
	// 设置pid文件
	if pinfo.pfile == "" {
		pinfo.pfile = pathtool.JoinPathFromHere(pinfo.name + ".pid")
	}
	return &Program{
		info:  info,
		cmds:  mapfx.NewUniqueStructSlice[*Command](true),
		pinfo: pinfo,
	}
}
