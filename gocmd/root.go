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
	"sort"
	"strconv"
	"strings"

	"github.com/xyzj/gopsu/mapfx"
)

type Program struct {
	info  *Info
	pinfo *ProcInfo
	cmds  *mapfx.UniqueSlice[*Command]
}

func (p *Program) printHelp() {
	s := make([]string, 0)
	maxname := 7
	x := p.cmds.Slice()
	sort.Slice(x, func(i, j int) bool {
		return x[i].Name < x[j].Name
	})
	for _, v := range x {
		if len(v.Name) > 7 {
			maxname = len(v.Name)
		}
	}

	for _, v := range x {
		s = append(s, fmt.Sprintf("  %-"+strconv.Itoa(maxname)+"s\t%s\n", v.Name, v.Descript))
	}
	println(fmt.Sprintf(`%s

%s

Usage:
  %s command [flags]

Available commands:
%s  version	show version info and exit.
  help		print this message. Or try use 'help <command>' to see more.

Flags:`, p.info.Title, p.info.Descript, p.pinfo.name, strings.Join(s, "")))
	flag.PrintDefaults()
}

// AddCommand add a command
func (p *Program) AddCommand(cmd *Command) *Program {
	if !p.cmds.Has(cmd) {
		p.cmds.Store(cmd)
	}
	return p
}

// BeforeStart 启动前执行的内容
func (p *Program) BeforeStart(f func()) *Program {
	p.pinfo.beforeStart = f
	return p
}

// AfterStop 收到停止信号后执行的内容
func (p *Program) AfterStop(f func()) *Program {
	p.pinfo.onSignalQuit = f
	return p
}

// // OnSignalQuit
// func (p *Program) OnSignalQuit(f func()) *Program {
// 	if f != nil {
// 		p.pinfo.onSignalQuit = f
// 	}
// 	return p
// }

// Execute Execute the given command, when no command is given, print help
func (p *Program) Execute() {
	if len(p.pinfo.params) == 0 { // no command, print help
		p.printHelp()
		os.Exit(0)
	}
	cmd := p.pinfo.params[0]
	if cmd == "version" { // print version message
		println(p.info.Ver)
		os.Exit(0)
	}
	if cmd == "help" { // print help message
		if len(p.pinfo.params) > 1 {
			for _, v := range p.cmds.Slice() {
				if v.Name == p.pinfo.params[1] {
					v.printHelp()
					os.Exit(0)
				}
			}
		}
		p.printHelp()
		os.Exit(0)
	}
	if p.pinfo.onSignalQuit == nil {
		p.pinfo.onSignalQuit = func() {}
	}
	if p.pinfo.beforeStart == nil {
		p.pinfo.beforeStart = func() {}
	}

	found := false
	code := 0
	for _, v := range p.cmds.Slice() {
		if v.Name == cmd { // 匹配到命令，开始执行
			found = true
			if len(p.pinfo.params) > 1 && p.pinfo.params[1] == "--help" {
				v.printHelp()
				os.Exit(0)
			}
			code = v.RunWithExitCode(p.pinfo)
			break
		}
	}
	if !found { // found nothing, print help
		p.printHelp()
	}
	if code != -1 {
		os.Exit(code)
	}
}

// ExecuteRun When no command is given, execute the run command instead of printing help
func (p *Program) ExecuteRun() {
	p.ExecuteDefault("run")
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
