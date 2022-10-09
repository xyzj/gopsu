package tools

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// String 内存地址转换[]byte
func String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Bytes 内存地址转换string
func Bytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			cap int
		}{s, len(s)},
	))
}

// IsExist file is exist or not
func IsExist(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
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

// GetExecDir get current file path
func GetExecDir() string {
	execdir, _ := filepath.EvalSymlinks(os.Args[0])
	execdir = filepath.Dir(execdir)
	if strings.Contains(execdir, "go-build") {
		execdir, _ = filepath.Abs(".")
	} else {
		execdir, _ = filepath.Abs(execdir)
	}
	return execdir
}
