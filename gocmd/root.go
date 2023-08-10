/*
Package gocmd A simplified console program framework that can quickly build a console program.

Usage：

	package main

	import (
		"github.com/xyzj/gopsu/gocmd"
	)

	func main() {
		gocmd.DefaultProgram(&gocmd.Info{
			Title:    "a test program",
			Descript: "this is a console program",
			Ver:      "v0.0.1",
		}).Execute()
		// do what you want...
	}
*/
package gocmd

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xyzj/gopsu/mapfx"
)

type Program struct {
	info  *Info
	pinfo *procInfo
	cmds  *mapfx.UniqueStructSlice[*Command]
}

func (p *Program) printHelp() {
	s := make([]string, 0)
	maxname := 7
	for _, v := range p.cmds.Slice() {
		if len(v.Name) > 7 {
			maxname = len(v.Name)
		}
	}
	for _, v := range p.cmds.Slice() {
		s = append(s, fmt.Sprintf("  %-"+strconv.Itoa(maxname)+"s\t%s\n", v.Name, v.Descript))
	}
	println(fmt.Sprintf(`%s
%s

Useage:
  %s command [flags]

Available commands:
%s  version	show version info and exit.
  help		print this message.

Flags:`, p.info.Title, p.info.Descript, p.pinfo.name, strings.Join(s, "")))
	flag.PrintDefaults()
}

// AddCommand add a command
func (p *Program) AddCommand(cmd *Command) error {
	if p.cmds.Has(cmd) {
		return fmt.Errorf("cmd %s already exists", cmd.Name)
	}
	cmd.pinfo = p.pinfo
	p.cmds.Store(cmd)
	return nil
}

// Execute Execute the given command, when no command is given, print help
func (p *Program) Execute() {
	// 只有1个，打印帮助
	if len(p.pinfo.params) == 0 {
		p.printHelp()
		os.Exit(0)
	}
	cmd := p.pinfo.params[0]
	if cmd == "version" {
		println(p.info.Ver)
		os.Exit(0)
	}
	found := false
	code := 0
	for _, v := range p.cmds.Slice() {
		if v.Name == cmd { // 匹配到命令，开始执行
			found = true
			code = v.RunWithExitCode(p.pinfo)
			break
		}
	}
	if !found {
		p.printHelp()
	}
	if code != -1 {
		os.Exit(code)
	}
}

// ExecuteDefault When no command is given, execute the specified command instead of printing help
func (p *Program) ExecuteDefault(cmd string) {
	if len(p.pinfo.params) == 0 {
		p.pinfo.params = []string{cmd}
	}
	if strings.HasPrefix(p.pinfo.params[0], "-") {
		x := []string{cmd}
		x = append(x, p.pinfo.params...)
		p.pinfo.params = x
	}
	p.Execute()
}
