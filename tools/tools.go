package tools

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

var (
	httpClient = &http.Client{
		// Timeout: time.Duration(time.Second * 60),
		Transport: &http.Transport{
			// DialContext: (&net.Dialer{
			// 	Timeout: time.Second * 2,
			// }).DialContext,
			// TLSHandshakeTimeout: time.Second * 2,
			IdleConnTimeout:     time.Second * 10,
			MaxConnsPerHost:     7777,
			MaxIdleConns:        1,
			MaxIdleConnsPerHost: 1,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
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

// DoRequestWithTimeout 发起请求
func DoRequestWithTimeout(req *http.Request, timeo time.Duration) (int, []byte, map[string]string, error) {
	// 处理头
	if req.Header.Get("Content-Type") == "" {
		switch req.Method {
		case "GET":
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		case "POST":
			req.Header.Set("Content-Type", "application/json")
		}
	}
	// 超时
	ctx, cancel := context.WithTimeout(context.Background(), timeo)
	defer cancel()
	// 请求
	start := time.Now()
	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return 502, nil, nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 502, nil, nil, err
	}
	end := time.Since(start).String()
	// 处理头
	h := make(map[string]string)
	h["resp_from"] = req.Host
	h["resp_duration"] = end
	for k := range resp.Header {
		h[k] = resp.Header.Get(k)
	}
	sc := resp.StatusCode
	return sc, b, h, nil
}

// DecodeString 解码混淆字符串，兼容python算法
func DecodeString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}
	s = ReverseString(SwapCase(s))
	if x := 4 - len(s)%4; x != 4 {
		for i := 0; i < x; i++ {
			s += "="
		}
	}
	if y, ex := base64.StdEncoding.DecodeString(s); ex == nil {
		var ns bytes.Buffer
		x := y[0]
		for k, v := range y {
			if k%2 != 0 {
				ns.WriteByte(v - x)
			}
		}
		return ns.String()
	}
	return ""
}

// ReverseString ReverseString
func ReverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

// SwapCase swap char case
func SwapCase(s string) string {
	var ns bytes.Buffer
	for _, v := range s {
		if v >= 65 && v <= 90 {
			ns.WriteString(string(v + 32))
		} else if v >= 97 && v <= 122 {
			ns.WriteString(string(v - 32))
		} else {
			ns.WriteString(string(v))
		}
	}
	return ns.String()
}
