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
	return filepath.Join(s...)
}

// GetExecDir get current file path
func GetExecDir() string {
	execdir, _ := filepath.EvalSymlinks(os.Args[0])
	if strings.Contains(execdir, "go-build") {
		execdir, _ = filepath.Abs(".")
	}
	return execdir
}

// SlicesUnion 求并集
func SlicesUnion(slice1, slice2 []string) []string {
	m := make(map[string]int)
	for _, v := range slice1 {
		if v == "" {
			continue
		}
		m[v]++
	}

	for _, v := range slice2 {
		if v == "" {
			continue
		}
		if _, ok := m[v]; !ok {
			slice1 = append(slice1, v)
		}
	}
	return slice1
}

// SlicesIntersect 求交集
func SlicesIntersect(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	for _, v := range slice1 {
		if v == "" {
			continue
		}
		m[v]++
	}

	for _, v := range slice2 {
		if v == "" {
			continue
		}
		if _, ok := m[v]; ok {
			nn = append(nn, v)
		}
	}
	return nn
}

// SlicesDifference 求差集 slice1-并集
func SlicesDifference(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	inter := SlicesIntersect(slice1, slice2)
	for _, v := range inter {
		if v == "" {
			continue
		}
		m[v]++
	}
	union := SlicesUnion(slice1, slice2)
	for _, v := range union {
		if v == "" {
			continue
		}
		if _, ok := m[v]; !ok {
			nn = append(nn, v)
		}
	}
	return nn
}
