// Package pathtool some path-related methods
package pathtool

import (
	"os"
	"path/filepath"
	"strings"
)

// SliceFlag 切片型参数，仅支持字符串格式
type SliceFlag []string

// String 返回参数
func (f *SliceFlag) String() string {
	return strings.Join(*f, ", ")
}

// Set 设置值
func (f *SliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// IsExist file is exist or not
func IsExist(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}

// GetExecFullpath get current file path
func GetExecFullpath() string {
	return JoinPathFromHere(GetExecName())
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
