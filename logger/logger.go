/*
Package logger 日志写入器，可设置是否自动依据日期以及文件大小滚动日志文件
*/
package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/xyzj/gopsu/loopfunc"
)

const (
	logformater     = "%s [%02d] %s"
	fileTimeFormat  = "060102"   // 日志文件命名格式
	maxFileSize     = 1048576000 // 1G
	shortTimeFormat = "15:04:05.000"
)

// OptLog OptLog
type OptLog struct {
	// Filename 日志文件名，不需要扩展名，会自动追加时间戳以及.log扩展名，为空时其他参数无效，仅输出到控制台
	Filename string
	// FileDir 日志存放目录
	FileDir string
	// AutoRoll 是否自动滚动日志文件，true-依据MaxDays和MaxSize自动切分日志文件，日志文件名会额外追加日期时间戳‘yymmdd’
	AutoRoll bool
	// MaxDays 日志最大保留天数，AutoRoll==true时有效
	MaxDays int
	// MaxSize 单个日志文件最大大小，AutoRoll==true时有效
	MaxSize int64
	// ZipFile 是否压缩旧日志文件，AutoRoll==true时有效
	ZipFile bool
}

// NewWriter 一个新的log写入器
//
// opt: 日志写入器配置
func NewWriter(opt *OptLog) io.Writer {
	if opt == nil {
		opt = &OptLog{}
	}
	if opt.AutoRoll { // 检查关联参数
		if opt.MaxDays < 1 {
			opt.MaxDays = 1
		}
		if opt.MaxSize > 0 && opt.MaxSize < 10485760 {
			opt.MaxSize = 10485760
		}
		if opt.MaxSize == 0 {
			opt.MaxSize = maxFileSize
		}
	}
	t := time.Now()
	mylog := &Writer{
		expired:      int64(opt.MaxDays)*24*60*60 - 10,
		fileMaxSize:  opt.MaxSize,
		fname:        opt.Filename,
		fileIndex:    0,
		rollfile:     opt.AutoRoll,
		fileDay:      t.Day(),
		fileHour:     t.Hour(),
		logDir:       opt.FileDir,
		chanWriteLog: make(chan []byte, 100),
		enablegz:     opt.ZipFile,
	}
	if opt.Filename != "" && opt.AutoRoll {
		ymd := t.Format(fileTimeFormat)
		for i := 1; i < 255; i++ {
			if !isExist(filepath.Join(mylog.logDir, fmt.Sprintf("%s.%v.%d.log", mylog.fname, ymd, i))) {
				mylog.fileIndex = byte(i) - 1
				break
			}
		}
		// for i := byte(255); i > 0; i-- {
		// 	if isExist(filepath.Join(mylog.logDir, fmt.Sprintf("%s.%v.%d.log", mylog.fname, ymd, i))) {
		// 		mylog.fileIndex = i
		// 		break
		// 	}
		// }
	}
	mylog.newFile()
	mylog.startWrite()
	return mylog
}

// Writer 自定义Writer
type Writer struct {
	rollfile     bool
	pathNow      string
	expired      int64
	fileMaxSize  int64
	fname        string
	nameNow      string
	nameOld      string
	fileIndex    byte
	logDir       string
	fileDay      int
	fileHour     int
	fno          *os.File
	enablegz     bool
	chanWriteLog chan []byte
	out          io.Writer
}

func (w *Writer) startWrite() {
	go loopfunc.LoopFunc(func(params ...interface{}) {
		tc := time.NewTicker(time.Minute * 10)
		for {
			select {
			case s := <-w.chanWriteLog:
				w.out.Write(s)
			case <-tc.C:
				if w.rollfile {
					w.rollingFileNoLock()
				}
			}
		}
	}, "log writer", nil)
}

func (w *Writer) Write(p []byte) (n int, err error) {
	b := toBytes(fmt.Sprintf("%s ", time.Now().Format(shortTimeFormat)))
	b = append(b, p...)
	ll := len(b)
	if b[ll-1] != 10 {
		b = append(b, 10)
	}
	w.chanWriteLog <- b
	return ll, nil
}

// 检查文件大小,返回是否需要切分文件
func (w *Writer) rolledWithFileSize() bool {
	if w.fileHour == time.Now().Hour() {
		return false
	}
	w.fileHour = time.Now().Hour()
	fs, ex := os.Stat(w.pathNow)
	if ex == nil {
		if fs.Size() > w.fileMaxSize {
			if w.fileIndex >= 255 {
				w.fileIndex = 0
			} else {
				w.fileIndex++
			}
			return true
		}
	}
	return false
}

func (w *Writer) rollingFileNoLock() bool {
	if time.Now().Day() == w.fileDay && !w.rolledWithFileSize() {
		return false
	}
	// t := time.Now()
	// w.nameNow = fmt.Sprintf("%s.%v.%d.log", w.fname, t.Format(fileTimeFormat), w.fileIndex)
	// // 比对文件名，若不同则重新设置io
	// if w.nameNow == w.nameOld {
	// 	return false
	// }
	// 创建新日志
	w.newFile()
	// 清理旧日志
	w.clearFile()

	return true
}

// 压缩旧日志
func (w *Writer) zipFile(s string) {
	if !w.enablegz || len(s) == 0 || !isExist(filepath.Join(w.logDir, s)) {
		return
	}
	// go func(s string) {
	// 	err := gopsu.ZIPFile(w.logDir, s, true)
	// 	if err != nil {
	// 		println("zip log file error: " + s + " " + err.Error())
	// 		return
	// 	}
	// }(s)
}

// 清理旧日志
func (w *Writer) clearFile() {
	// 若未设置超时，则不清理
	if !w.rollfile || w.expired <= 0 {
		return
	}
	go func() {
		defer func() { recover() }()
		// 遍历文件夹
		lstfno, ex := ioutil.ReadDir(w.logDir)
		if ex != nil {
			println(fmt.Sprintf("clear log files error: %s", ex.Error()))
			return
		}
		t := time.Now()
		for _, fno := range lstfno {
			if fno.IsDir() || !strings.Contains(fno.Name(), w.fname) { // 忽略目录，不含日志名的文件，以及当前文件
				continue
			}
			// 比对文件生存期
			if t.Unix()-fno.ModTime().Unix() >= w.expired {
				os.Remove(filepath.Join(w.logDir, fno.Name()))
			}
		}
	}()
}

// 创建新日志文件
func (w *Writer) newFile() {
	if w.fname == "" {
		w.out = os.Stdout
		return
	}
	if w.rollfile {
		t := time.Now()
		if w.fileDay != t.Day() {
			w.fileDay = t.Day()
			w.fileIndex = 0
		}
		w.nameNow = fmt.Sprintf("%s.%v.%d.log", w.fname, t.Format(fileTimeFormat), w.fileIndex)
	} else {
		w.nameNow = fmt.Sprintf("%s.log", w.fname)
	}
	if w.nameOld == w.nameNow {
		return
	}
	// 关闭旧fno
	if w.fno != nil {
		w.fno.Close()
	}
	w.pathNow = filepath.Join(w.logDir, w.nameNow)
	// 直接写入当日日志
	var err error
	w.fno, err = os.OpenFile(w.pathNow, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
	if err != nil {
		ioutil.WriteFile("logerr.log", toBytes("log file open error: "+err.Error()), 0664)
		w.out = os.Stdout
	} else {
		w.out = w.fno
	}
	// 判断是否压缩旧日志
	if w.enablegz {
		w.zipFile(w.nameOld)
	}
	w.out.Write([]byte{10})
}
func isExist(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}

// toBytes 内存地址转换string
func toBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			cap int
		}{s, len(s)},
	))
	// x := (*[2]uintptr)(unsafe.Pointer(&s))
	// h := [3]uintptr{x[0], x[1], x[1]}
	// return *(*[]byte)(unsafe.Pointer(&h))
}
func getExecDir() string {
	a, _ := os.Executable()
	execdir := filepath.Dir(a)
	if strings.Contains(execdir, "go-build") {
		execdir, _ = filepath.Abs(".")
	}
	return execdir
}
