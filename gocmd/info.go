package gocmd

import (
	"encoding/json"
	"os"
	"strings"
)

// Info program information
type Info struct {
	// Ver program version
	Ver string
	// Title program title
	Title string
	// Descript program descript message
	Descript string
}

type procInfo struct {
	// os.Args[1:]
	params []string
	// "run"+os.Args[2:]
	args []string
	// fullpath of the program
	exec string
	// filepath.Base(exec)
	name string
	// filepath.Dir(exec)
	dir string
	// the file save the pid
	pfile string
	// pid value
	pid string
}

type version struct {
	Dependencies []string `json:"deps,omitempty"`
	Name         string   `json:"name,omitempty"`
	Version      string   `json:"version,omitempty"`
	GoVersion    string   `json:"go_version,omitempty"`
	BuildDate    string   `json:"build_date,omitempty"`
	BuildOS      string   `json:"build_os,omitempty"`
	CodeBy       string   `json:"code_by,omitempty"`
	StartWith    string   `json:"start_with,omitempty"`
}

// VersionInfo show something
//
// name: program name
// ver: program version
// gover: golang version
// buildDate: build datetime
// buildOS: platform info
// auth: auth name
func VersionInfo(name, ver, gover, buildDate, buildOS, auth string) string {
	b, _ := json.MarshalIndent(&version{
		Name:      name,
		Version:   ver,
		GoVersion: gover,
		BuildDate: buildDate,
		BuildOS:   buildOS,
		CodeBy:    auth,
		StartWith: strings.Join(os.Args[1:], " "),
	}, "", "  ")
	return string(b)
}
