package mxgo

import (
	"bytes"
	"compress/zlib"
	"container/list"
	b64 "encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// OSNAME from runtime
	OSNAME = runtime.GOOS
	// OSARCH from runtime
	OSARCH = runtime.GOARCH
)

// Queue queue for go
type Queue struct {
	Q *list.List
}

// Put put data to the end of the queue
func (mq *Queue) Put(value interface{}) {
	mq.Q.PushBack(value)
}

// Get get data from front
func (mq *Queue) Get() interface{} {
	if mq.Q.Len() == 0 {
		return nil
	}
	e := mq.Q.Front()
	if e != nil {
		mq.Q.Remove(e)
		return e.Value
	}
	return nil
}

// Len get queue len
func (mq *Queue) Len() int64 {
	return int64(mq.Q.Len())
}

// Empty check if empty
func (mq *Queue) Empty() bool {
	return mq.Q.Len() == 0
}

// Clean clean the queue
func (mq *Queue) Clean() {
	var n *list.Element
	for e := mq.Q.Front(); e != nil; e = n {
		n = e.Next()
		mq.Q.Remove(e)
	}
}

// MakeRuntimeDirs make conf,log,cache dirs
// Args：
//	rootpath： 输入路径
// return：
// 	conf，log，cache三个文件夹的完整路径
func MakeRuntimeDirs(rootpath string) (string, string, string) {
	var basepath string
	if strings.Compare(rootpath, ".") == 0 {
		basepath = GetExecDir()
	} else {
		basepath = rootpath
	}
	os.MkdirAll(filepath.Join(basepath, "..", "conf"), 0775)
	os.MkdirAll(filepath.Join(basepath, "..", "log"), 0775)
	os.MkdirAll(filepath.Join(basepath, "..", "cache"), 0775)
	return filepath.Join(basepath, "..", "conf"), filepath.Join(basepath, "..", "log"), filepath.Join(basepath, "..", "cache")
}

//String2Bytes convert hex-string to []byte
// Args:
// 	data: 输入字符串
// 	sep： 用于分割字符串的分割字符
// Return:
// 	字节切片
func String2Bytes(data string, sep string) []byte {
	var z []byte
	a := strings.Split(data, sep)
	z = make([]byte, len(a))
	for k, v := range a {
		b, _ := strconv.ParseUint(v, 16, 8)
		z[k] = byte(b)
	}
	return z
}

// Bytes2String convert []byte to hex-string
// Args:
// 	data: 输入字节切片
// 	sep： 用于分割字符串的分割字符
// Return:
// 	字符串
func Bytes2String(data []byte, sep string) string {
	a := make([]string, len(data))
	for k, v := range data {
		a[k] = fmt.Sprintf("%02x", v)
	}
	return strings.Join(a, sep)
}

// String2Int64 convert string 2 int64
// Args:
// 	s: 输入字符串
// 	t: 返回数值进制
// Return：
// 	int64
func String2Int64(s string, t int) int64 {
	x, _ := strconv.ParseInt(s, t, 0)
	return x
}

// String2Int32 convert string 2 int32
// Args:
// 	s: 输入字符串
// 	t: 返回数值进制
// Return：
// 	int64
func String2Int32(s string, t int) int32 {
	x, _ := strconv.ParseInt(s, t, 0)
	return int32(x)
}

//StringSlice2Int8 convert string Slice 2 int8
func StringSlice2Int8(bs []string) byte {
	return String2Int8(strings.Join(bs, ""), 2)
}

// String2Int8 convert string 2 int8
func String2Int8(s string, t int) byte {
	x, _ := strconv.ParseInt(s, t, 0)
	return byte(x)
}

// CheckLrc check lrc data
func CheckLrc(d []byte) bool {
	rowdata := d[:len(d)-1]
	lrcdata := d[len(d)-1]

	c := CountLrc(&rowdata)
	if c == lrcdata {
		return true
	}
	return false
}

// CountLrc count lrc data
func CountLrc(data *[]byte) byte {
	a := byte(0)
	for _, v := range *data {
		a ^= v
	}
	return a
}

// CheckCrc16VB check crc16 data
func CheckCrc16VB(d []byte) bool {
	rowdata := d[:len(d)-2]
	crcdata := d[len(d)-2:]

	c := CountCrc16VB(&rowdata)
	if c[0] == crcdata[0] && c[1] == crcdata[1] {
		return true
	}
	return false
}

// CountCrc16VB count crc16 as vb6 do
func CountCrc16VB(data *[]byte) []byte {
	var z = make([]byte, 0)
	crc16lo := byte(0xFF)
	crc16hi := byte(0xFF)
	cl := byte(0x01)
	ch := byte(0xa0)
	for _, v := range *data {
		crc16lo ^= v
		for i := 0; i < 8; i++ {
			savehi := crc16hi
			savelo := crc16lo
			crc16hi /= 2
			crc16lo /= 2
			if (savehi & 0x01) == 0x01 {
				crc16lo ^= 0x80
			}
			if (savelo & 0x01) == 0x01 {
				crc16hi ^= ch
				crc16lo ^= cl
			}
		}
	}
	z = append(z, byte(crc16lo), byte(crc16hi))
	return z
}

// IPUint2String change ip int64 data to string format
func IPUint2String(ipnr uint) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ipnr>>24)&0xFF, (ipnr>>16)&0xFF, (ipnr>>8)&0xFF, ipnr&0xFF)
}

// IPInt642String change ip int64 data to string format
func IPInt642String(ipnr int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ipnr)&0xFF, (ipnr>>8)&0xFF, (ipnr>>16)&0xFF, (ipnr>>24)&0xFF)
}

// IPInt642Bytes change ip int64 data to string format
func IPInt642Bytes(ipnr int64) []byte {
	return []byte{byte((ipnr) & 0xFF), byte((ipnr >> 8) & 0xFF), byte((ipnr >> 16) & 0xFF), byte((ipnr >> 24) & 0xFF)}
}

// IPUint2Bytes change ip int64 data to string format
func IPUint2Bytes(ipnr int64) []byte {
	return []byte{byte((ipnr >> 24) & 0xFF), byte((ipnr >> 16) & 0xFF), byte((ipnr >> 8) & 0xFF), byte((ipnr) & 0xFF)}
}

// IP2Uint change ip string data to int64 format
func IP2Uint(ipnr string) uint {
	// ex := errors.New("wrong ip address format")
	bits := strings.Split(ipnr, ".")
	if len(bits) != 4 {
		return 0
	}
	var intip uint
	for k, v := range bits {
		i, ex := strconv.Atoi(v)
		if ex != nil || i > 255 || i < 0 {
			return 0
		}
		intip += uint(i) << uint(8*(3-k))
	}
	return intip
}

// IP2Int64 change ip string data to int64 format
func IP2Int64(ipnr string) int64 {
	// ex := errors.New("wrong ip address format"
	bits := strings.Split(ipnr, ".")
	if len(bits) != 4 {
		return 0
	}
	var intip uint
	for k, v := range bits {
		i, ex := strconv.Atoi(v)
		if ex != nil || i > 255 || i < 0 {
			return 0
		}
		intip += uint(i) << uint(8*(k))
	}
	return int64(intip)
}

// IsExist file is exist or not
func IsExist(p string) bool {
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}

// GetExecDir get current file path
func GetExecDir() string {
	a, _ := os.Executable()
	execdir := filepath.Dir(a)
	if strings.Contains(execdir, "command-line-arguments") {
		execdir, _ = filepath.Abs(".")
	}
	return execdir
}

//SplitDateTime SplitDateTime
func SplitDateTime(sdt int64) (y, m, d, h, mm, s, wd byte) {
	if sdt == 0 {
		sdt = time.Now().Unix()
	}
	tm := time.Unix(sdt, 0)
	stm := tm.Format("2006-01-02 15:04:05 Mon")
	dt := strings.Split(stm, " ")
	dd := strings.Split(dt[0], "-")
	tt := strings.Split(dt[1], ":")
	return byte(String2Int32(dd[0], 10) - 2000),
		String2Int8(dd[1], 10),
		String2Int8(dd[2], 10),
		String2Int8(tt[0], 10),
		String2Int8(tt[1], 10),
		String2Int8(tt[2], 10),
		byte(tm.Weekday())
}

// Stamp2Time convert stamp to datetime string
func Stamp2Time(t int64) string {
	tm := time.Unix(t, 0)
	return tm.Format("2006-01-02 15:04:05")
}

// Time2Stamp convert datetime string to stamp
func Time2Stamp(t string) int64 {
	loc, _ := time.LoadLocation("Local")
	tm, ex := time.ParseInLocation("2006-01-02 15:04:05", t, loc)
	// tm, ex := time.Parse("2006-01-02 15:04:05", t)
	if ex != nil {
		return 0
	}
	return tm.Unix()
}

// SwitchStamp switch stamp format between unix and c#
func SwitchStamp(t int64) int64 {
	y := int64(621356256000000000)
	z := int64(10000000)
	if t > y {
		return (t - y) / z
	}
	return t*z + y
}

// Byte2Bytes int8 to bytes
func Byte2Bytes(v byte, reverse bool) []byte {
	s := fmt.Sprintf("%08b", v)
	if reverse {
		s = ReverseString(s)
	}
	b := make([]byte, 0)
	for _, v := range s {
		if v == 48 {
			b = append(b, 0)
		} else {
			b = append(b, 1)
		}
	}
	return b
}

// Byte2Int32s int8 to int32 list
func Byte2Int32s(v byte, reverse bool) []int32 {
	s := fmt.Sprintf("%08b", v)
	if reverse {
		s = ReverseString(s)
	}
	b := make([]int32, 0)
	for _, v := range s {
		if v == 48 {
			b = append(b, 0)
		} else {
			b = append(b, 1)
		}
	}
	return b
}

// Bcd2Int8 bcd to int
func Bcd2Int8(v byte) byte {
	return ((v&0xf0)>>4)*10 + (v & 0x0f)
}

// Int82Bcd int to bcd
func Int82Bcd(v byte) byte {
	return ((v / 10) << 4) | (v % 10)
}

// ReverseString ReverseString
func ReverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

// DecodeString 解码混淆字符串，兼容python算法
func DecodeString(s string) string {
	s = SwapCase(s)
	var ns bytes.Buffer
	ns.Write([]byte{120, 156})
	if x := 4 - len(s)%4; x != 4 {
		for i := 0; i < x; i++ {
			s += "="
		}
	}
	if y, ex := b64.StdEncoding.DecodeString(s); ex == nil {
		x := String2Int8(string(y[0])+string(y[1]), 0)
		z := y[2:]
		for i := len(z) - 1; i >= 0; i-- {
			if z[i] >= x {
				ns.WriteByte(z[i] - x)
			} else {
				ns.WriteByte(byte(int(z[i]) + 256 - int(x)))
			}
		}
		return ReverseString(string(DoZlibUnCompress(ns.Bytes())))
	} else {
		return "You screwed up."
	}
}

// DoZlibUnCompress zlib uncompress
func DoZlibUnCompress(src []byte) []byte {
	b := bytes.NewReader(src)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	io.Copy(&out, r)
	return out.Bytes()
}

// DoZlibCompress zlib compress
func DoZlibCompress(src []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

// SwapCase swap char case
func SwapCase(s string) string {
	var ns bytes.Buffer
	for _, v := range s {
		// println(v, string(v))
		if v >= 65 && v <= 90 {
			ns.WriteString(string(int(v) + 32))
		} else if v >= 97 && v <= 122 {
			ns.WriteString(string(int(v) - 32))
		} else {
			ns.WriteString(string(v))
		}
	}
	return ns.String()
}

// VersionInfo show something
func VersionInfo(p, v, gv, bd, a string) string {
	return fmt.Sprintf("\n%s \nVersion:\t%s\nGo version:\t%s\nBuild date:\t%s\nCode by:\t%s", p, v, gv, bd, a)
}
