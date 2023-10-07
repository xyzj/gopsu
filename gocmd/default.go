package gocmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
		info = &Info{}
	}
	if info.Title == "" {
		info.Title = "A general program startup framework"
	}
	if info.Descript == "" {
		info.Descript = "can run program in the background"
	}
	if info.Ver == "" {
		info.Ver = "0.0.1"
	}
	params := os.Args
	pinfo := &ProcInfo{
		params: make([]string, 0),
		Args:   make([]string, 0),
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
	// 设置pid文件
	pinfo.pfile = os.Getenv(fmt.Sprintf("%s_PID_FILE", strings.ToUpper(pathtool.GetExecNameWithoutExt())))
	if pinfo.pfile == "" {
		pinfo.pfile = pathtool.JoinPathFromHere(pinfo.name + ".pid")
	}
	notparseflag, _ := strconv.ParseBool(os.Getenv(fmt.Sprintf("%s_NOT_PARSE_FLAG", strings.ToUpper(pathtool.GetExecNameWithoutExt()))))
	// 处理参数
	idx := 0
	for k, v := range params {
		if !strings.HasPrefix(v, "-") || v == "--help" {
			continue
		}
		idx = k
		break
	}
	pinfo.Args = params[idx:]
	if !notparseflag {
		flag.CommandLine.Parse(params[idx:])
	}
	return &Program{
		info:  info,
		cmds:  mapfx.NewUniqueSlice[*Command](),
		pinfo: pinfo,
	}
}
