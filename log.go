package gopsu

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	logDebug    = 10
	logInfo     = 20
	logWarning  = 30
	logError    = 40
	logSystem   = 90
	maxFileLife = 15*24*60*60 - 10
	maxFileSize = 1048576000 // 1G
)

var (
	asyncCache = 1000
)

// Logger 日志接口
type Logger interface {
	Debug(msgs string)
	Info(msgs string)
	Warning(msgs string)
	Error(msgs string)
	System(msgs string)
	DebugFormat(f string, msgs ...interface{})
	InfoFormat(f string, msgs ...interface{})
	WarningFormat(f string, msgs ...interface{})
	ErrorFormat(f string, msgs ...interface{})
	SystemFormat(f string, msgs ...interface{})
}

// NilLogger 空日志
type NilLogger struct{}

// Debug Debug
func (l *NilLogger) Debug(msgs string) {}

// Info Info
func (l *NilLogger) Info(msgs string) {}

// Warning Warning
func (l *NilLogger) Warning(msgs string) {}

// Error Error
func (l *NilLogger) Error(msgs string) {}

// System System
func (l *NilLogger) System(msgs string) {}

// DebugFormat Debug
func (l *NilLogger) DebugFormat(f string, msg ...interface{}) {}

// InfoFormat Info
func (l *NilLogger) InfoFormat(f string, msg ...interface{}) {}

// WarningFormat Warning
func (l *NilLogger) WarningFormat(f string, msg ...interface{}) {}

// ErrorFormat Error
func (l *NilLogger) ErrorFormat(f string, msg ...interface{}) {}

// SystemFormat System
func (l *NilLogger) SystemFormat(f string, msg ...interface{}) {}

// StdLogger 空日志
type StdLogger struct{}

// Debug Debug
func (l *StdLogger) Debug(msgs string) {
	println(msgs)
}

// Info Info
func (l *StdLogger) Info(msgs string) {
	println(msgs)
}

// Warning Warning
func (l *StdLogger) Warning(msgs string) {
	println(msgs)
}

// Error Error
func (l *StdLogger) Error(msgs string) {
	println(msgs)
}

// System System
func (l *StdLogger) System(msgs string) {
	println(msgs)
}

// DebugFormat Debug
func (l *StdLogger) DebugFormat(f string, msg ...interface{}) {
	println(fmt.Sprintf(f, msg...))
}

// InfoFormat Info
func (l *StdLogger) InfoFormat(f string, msg ...interface{}) {
	println(fmt.Sprintf(f, msg...))
}

// WarningFormat Warning
func (l *StdLogger) WarningFormat(f string, msg ...interface{}) {
	println(fmt.Sprintf(f, msg...))
}

// ErrorFormat Error
func (l *StdLogger) ErrorFormat(f string, msg ...interface{}) {
	println(fmt.Sprintf(f, msg...))
}

// SystemFormat System
func (l *StdLogger) SystemFormat(f string, msg ...interface{}) {
	println(fmt.Sprintf(f, msg...))
}

// MxLog mx log
type MxLog struct {
	fileFullPath  string
	fileSize      int64
	fileMaxLife   int64
	fileMaxSize   int64
	fileName      string
	fileNameNow   string
	fileNameOld   string
	fileIndex     byte
	fileDir       string
	fileDay       int
	fileHour      int
	fno           *os.File
	logLevel      int
	enablegz      bool
	err           error
	fileLock      sync.RWMutex
	chanWrite     chan *logMessage
	chanClose     chan bool
	writeAsync    bool
	asyncLock     sync.WaitGroup
	chanWatcher   chan string
	defaultWriter io.Writer
}

type logMessage struct {
	msg   string
	level int
}

func (l *MxLog) getFileSize() int64 {
	f, ex := os.Stat(l.fileFullPath)
	if ex != nil {
		l.fileSize = 0
	}
	l.fileSize = f.Size()
	return l.fileSize
}

// SetMaxFileLife set max day log file keep
func (l *MxLog) SetMaxFileLife(c int64) {
	l.fileMaxLife = c*24*60*60 - 10
}

// SetMaxFileCount [Discard] use SetMaxFileLife() instead
func (l *MxLog) SetMaxFileCount(c uint16) {
	l.SetMaxFileLife(int64(c))
}

// SetMaxFileSize set max log file size in M
func (l *MxLog) SetMaxFileSize(c int64) {
	l.fileMaxSize = c * 1024000
}

// DefaultWriter DefaultWriter
func (l *MxLog) DefaultWriter() io.Writer {
	return l.defaultWriter
}

// SetLogLevel set file & console log level
func (l *MxLog) SetLogLevel(loglevel int, conlevel ...int) {
	l.logLevel = loglevel

	if l.logLevel <= 10 {
		l.defaultWriter = io.MultiWriter(l.fno, os.Stdout)
	} else {
		l.defaultWriter = io.MultiWriter(l.fno)
	}
}

// SetAsync 设置异步写入参数
func (l *MxLog) SetAsync(c int) {
	if c <= 0 {
		l.writeAsync = false
	}
	l.writeAsync = true
}

func (l *MxLog) coreWatcher() {
	closeme := false
	for {
		if closeme {
			break
		}
		select {
		case n := <-l.chanWatcher:
			time.Sleep(100 * time.Millisecond)
			switch n {
			case "mxlog":
				go l.writeLogAsync()
			}
		}
	}
}

// StartWriteLog StartWriteLog
func (l *MxLog) writeLogAsync() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				ioutil.WriteFile(fmt.Sprintf("crash-log-%s.log", time.Now().Format("20060102150405")), []byte(fmt.Sprintf("%v", err.(error))), 0644)
				time.Sleep(300 * time.Millisecond)
			}
			l.chanWatcher <- "mxlog"
		}()
		closeme := false
		// t := time.NewTicker(time.Hour)
		for {
			if closeme {
				break
			}
			select {
			case msg := <-l.chanWrite:
				l.writeLog(msg.msg, msg.level)
				// case <-t.C:
				// fs, ex := os.Stat(l.fileFullPath)
				// if ex == nil {
				// 	if fs.Size() > 1048576000 {
				// 		if l.fileIndex >= 255 {
				// 			l.fileIndex = 0
				// 		} else {
				// 			l.fileIndex++
				// 		}
				// 	}
				// }
			}
		}
	}()
}

// WriteLog 写日志
func (l *MxLog) WriteLog(msg string, level int) {
	l.writeLog(msg, level)
}

func (l *MxLog) writeLog(msg string, level int, lock ...bool) {
	if l.writeAsync {
		l.rollingFile()
	} else {
		l.rollingFileNoLock()
	}

	if level >= l.logLevel {
		s := fmt.Sprintf("%s [%02d] %s", time.Now().Format(ShortTimeFormat), level, msg)
		fmt.Fprintln(l.defaultWriter, s)
		if level >= 40 && l.logLevel >= 20 {
			println(s)
		}
	}
}

// Debug writelog with level 10
func (l *MxLog) Debug(msgs ...string) {
	msg := strings.Join(msgs, ",")
	if l.writeAsync {
		l.chanWrite <- &logMessage{
			msg:   msg,
			level: logDebug,
		}
	} else {
		l.writeLog(msg, logDebug, true)
	}
}

// Info writelog with level 20
func (l *MxLog) Info(msgs ...string) {
	msg := strings.Join(msgs, ",")
	if l.writeAsync {
		l.chanWrite <- &logMessage{
			msg:   msg,
			level: logInfo,
		}
	} else {
		l.writeLog(msg, logInfo, true)
	}
}

// Warning writelog with level 30
func (l *MxLog) Warning(msgs ...string) {
	msg := strings.Join(msgs, ",")
	if l.writeAsync {
		l.chanWrite <- &logMessage{
			msg:   msg,
			level: logWarning,
		}
	} else {
		l.writeLog(msg, logWarning, true)
	}
}

// Error writelog with level 40
func (l *MxLog) Error(msgs ...string) {
	msg := strings.Join(msgs, ",")
	if l.writeAsync {
		l.chanWrite <- &logMessage{
			msg:   msg,
			level: logError,
		}
	} else {
		l.writeLog(msg, logError, true)
	}
	// _, fn, lno, _ := runtime.Caller(1)
	// go l.writeLog(fmt.Sprintf("[%s:%d] %s", filepath.Base(fn), lno, msg), logError)
}

// System writelog with level 90
func (l *MxLog) System(msgs ...string) {
	msg := strings.Join(msgs, ",")
	if l.writeAsync {
		l.chanWrite <- &logMessage{
			msg:   msg,
			level: logSystem,
		}
	} else {
		l.writeLog(msg, logSystem, true)
	}
}

// CurrentFileSize current file size
func (l *MxLog) CurrentFileSize() int64 {
	return l.fileSize
}

// Close close logger
func (l *MxLog) Close() error {
	if l.writeAsync {
		l.chanClose <- true
	}
	return l.fno.Close()
}

// EnableGZ EnableGZ
func (l *MxLog) EnableGZ(b bool) {
	l.enablegz = b
}

// InitNewLogger [Discard] use NewLogger() instead
func InitNewLogger(p string) *MxLog {
	return NewLogger(filepath.Dir(p), filepath.Base(p))
}

// NewLogger init logger
//   d: log file path
//   f: log file name
func NewLogger(d, f string) *MxLog {
	t := time.Now()
	mylog := &MxLog{
		fileMaxLife: maxFileLife,
		fileMaxSize: maxFileSize,
		fileName:    f,
		fileIndex:   0,
		fileDay:     t.Day(),
		fileHour:    t.Hour(),
		fileDir:     d,
		logLevel:    logDebug,
		chanWrite:   make(chan *logMessage, 5000),
		chanClose:   make(chan bool, 2),
		chanWatcher: make(chan string, 2),
		writeAsync:  false,
		enablegz:    true,
	}

	for i := byte(0); i < 255; i++ {
		if IsExist(filepath.Join(mylog.fileDir, fmt.Sprintf("%s.%v.%d.log", mylog.fileName, t.Format(FileTimeFormat), i))) ||
			IsExist(filepath.Join(mylog.fileDir, fmt.Sprintf("%s.%v.%d.log.zip", mylog.fileName, t.Format(FileTimeFormat), i))) {
			mylog.fileIndex = i
		} else {
			break
		}
	}

	go mylog.coreWatcher()
	go mylog.writeLogAsync()
	mylog.newFile()

	return mylog
}

// 检查文件大小,返回是否需要切分文件
func (l *MxLog) rolledWithFileSize() bool {
	if l.fileHour == time.Now().Hour() {
		return false
	}
	l.fileHour = time.Now().Hour()
	fs, ex := os.Stat(l.fileFullPath)
	if ex == nil {
		if fs.Size() > l.fileMaxSize {
			if l.fileIndex >= 255 {
				l.fileIndex = 0
			} else {
				l.fileIndex++
			}
			return true
		}
	}
	return false
}

func (l *MxLog) rollingFileNoLock() bool {
	t := time.Now()
	l.rolledWithFileSize()
	l.fileNameNow = fmt.Sprintf("%s.%v.%d.log", l.fileName, t.Format(FileTimeFormat), l.fileIndex)
	// 比对文件名，若不同则重新设置io
	if l.fileNameNow == l.fileNameOld {
		return false
	}
	// 关闭旧fno
	l.fno.Close()
	// 创建新日志
	l.newFile()
	// 清理旧日志
	l.clearFile()

	return true
}

// 按日期切分文件
func (l *MxLog) rollingFile() bool {
	l.fileLock.Lock()
	defer l.fileLock.Unlock()

	return l.rollingFileNoLock()
}

// 压缩旧日志
func (l *MxLog) zipFile(s string) {
	if !l.enablegz || len(s) == 0 || !IsExist(filepath.Join(l.fileDir, s)) {
		return
	}
	go func() {
		ZIPFile(l.fileDir, s, true)
		// // 删除已压缩的旧日志
		// err := os.Remove(filepath.Join(l.fileDir, s))
		// if err != nil {
		// 	ioutil.WriteFile(fmt.Sprintf("logcrash.%d.log", time.Now().Unix()), []byte("del old file:"+s+" "+err.Error()), 0664)
		// }
	}()
}

// 清理旧日志
func (l *MxLog) clearFile() {
	// 若未设置超时，则不清理
	if l.fileMaxLife == 0 {
		return
	}
	go func() {
		defer func() { recover() }()
		// 遍历文件夹
		lstfno, ex := ioutil.ReadDir(l.fileDir)
		if ex != nil {
			println(fmt.Sprintf("clear log files error: %s", ex.Error()))
		}
		t := time.Now()
		for _, fno := range lstfno {
			if fno.IsDir() || !strings.Contains(fno.Name(), l.fileName) { // 忽略目录，不含日志名的文件，以及当前文件
				continue
			}
			// 比对文件生存期
			if t.Unix()-fno.ModTime().Unix() >= l.fileMaxLife {
				os.Remove(filepath.Join(l.fileDir, fno.Name()))
			}
		}
	}()
}

// 创建新日志文件
func (l *MxLog) newFile() {
	t := time.Now()
	if l.fileDay != t.Day() {
		l.fileDay = t.Day()
		l.fileIndex = 0
	}
	l.fileNameNow = fmt.Sprintf("%s.%v.%d.log", l.fileName, t.Format(FileTimeFormat), l.fileIndex)
	l.fileFullPath = filepath.Join(l.fileDir, l.fileNameNow)
	// 直接写入当日日志
	// 打开文件
	l.fno, l.err = os.OpenFile(l.fileFullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if l.err != nil {
		println("Log file open error: " + l.err.Error())
	}
	if l.logLevel <= 10 {
		l.defaultWriter = io.MultiWriter(l.fno, os.Stdout)
	} else {
		l.defaultWriter = io.MultiWriter(l.fno)
	}
	// 判断是否压缩旧日志
	if l.enablegz {
		l.zipFile(l.fileNameOld)
	}
	l.fileNameOld = l.fileNameNow
}
