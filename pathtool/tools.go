// Package pathtool some path-related methods
package pathtool

import (
	"os"
	"path/filepath"
	"strings"
)

// IsExist file is exist or not
func IsExist(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}

// GetExecDir get current file path
func GetExecDir() string {
	a, _ := os.Executable()
	execdir := filepath.Dir(a)
	if strings.Contains(execdir, "go-build") {
		execdir, _ = filepath.Abs(".")
	}
	return execdir
}

// GetExecName 获取可执行文件的名称
func GetExecName() string {
	exe, _ := os.Executable()
	if exe == "" {
		return ""
	}
	return filepath.Base(exe)
}

// GetExecNameWithoutExt 获取可执行文件的名称,去除扩展名
func GetExecNameWithoutExt() string {
	name := GetExecName()
	return strings.ReplaceAll(name, filepath.Ext(name), "")
}

// JoinPathFromHere 从程序执行目录开始拼接路径
func JoinPathFromHere(path ...string) string {
	s := []string{GetExecDir()}
	s = append(s, path...)
	sp := filepath.Join(s...)
	p, err := filepath.Abs(sp)
	if err != nil {
		return sp
	}
	return p
}
