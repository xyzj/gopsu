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
	logformater = "%s [%02d] %s"
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
	DefaultWriter() io.Writer
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

// DefaultWriter 返回日志Writer
func (l *NilLogger) DefaultWriter() io.Writer { return nil }

// StdLogger 空日志
type StdLogger struct{}

// Debug Debug
func (l *StdLogger) Debug(msgs string) {
	l.writeLog(msgs, 10)
}

// Info Info
func (l *StdLogger) Info(msgs string) {
	l.writeLog(msgs, 20)
}

// Warning Warning
func (l *StdLogger) Warning(msgs string) {
	l.writeLog(msgs, 30)
}

// Error Error
func (l *StdLogger) Error(msgs string) {
	l.writeLog(msgs, 40)
}

// System System
func (l *StdLogger) System(msgs string) {
	l.writeLog(msgs, 90)
}

// DebugFormat Debug
func (l *StdLogger) DebugFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), 10)
}

// InfoFormat Info
func (l *StdLogger) InfoFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), 20)
}

// WarningFormat Warning
func (l *StdLogger) WarningFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), 30)
}

// ErrorFormat Error
func (l *StdLogger) ErrorFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), 40)
}

// SystemFormat System
func (l *StdLogger) SystemFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), 90)
}

// DefaultWriter 返回日志Writer
func (l *StdLogger) DefaultWriter() io.Writer { return os.Stdout }

func (l *StdLogger) writeLog(msg string, level int) {
	println(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), level, msg))
}

// MxLog mx log
type MxLog struct {
	pathNow string
	// fileSize      int64
	expired       int64
	fileMaxSize   int64
	fname         string
	nameNow       string
	nameOld       string
	fileIndex     byte
	logDir        string
	fileDay       int
	fileHour      int
	fno           *os.File
	logLevel      int
	enablegz      bool
	err           error
	fileLock      sync.RWMutex
	chanWriteLog  chan string
	out           io.Writer
	logClassified bool
	cWorker       *CryptoWorker
}

// type logMessage struct {
// 	msg   string
// 	level int
// }

// SetMaxFileLife set max day log file keep
// func (l *MxLog) SetMaxFileLife(c int64) {
// 	l.expired = c*24*60*60 - 10
// }

// SetMaxFileCount [Discard] use SetMaxFileLife() instead
// func (l *MxLog) SetMaxFileCount(c uint16) {
// 	l.SetMaxFileLife(int64(c))
// }

// SetMaxFileSize set max log file size in M
// func (l *MxLog) SetMaxFileSize(c int64) {
// 	l.fileMaxSize = c * 1024000
// }

// DefaultWriter out
func (l *MxLog) DefaultWriter() io.Writer {
	return l.out
}

// SetLogLevel set file & console log level
// func (l *MxLog) SetLogLevel(loglevel int, conlevel ...int) {
// 	l.logLevel = loglevel

// 	if l.logLevel <= 10 {
// 		l.out = io.MultiWriter(l.fno, os.Stdout)
// 	} else {
// 		l.out = io.MultiWriter(l.fno)
// 	}
// }

// WriteLog 写日志
func (l *MxLog) WriteLog(msg string, level int) {
	l.writeLog(msg, level)
}

func (l *MxLog) writeLog(msg string, level int, lock ...bool) {
	if l.fname != "" && !IsExist(l.pathNow) {
		l.fno.Close()
		// 打开文件
		l.fno, l.err = os.OpenFile(l.pathNow, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
		if l.err != nil {
			ioutil.WriteFile("logerr.log", []byte("Log file reopen error: "+l.err.Error()), 0664)
			l.out = io.MultiWriter(os.Stdout)
		} else {
			if l.logLevel <= 10 {
				l.out = io.MultiWriter(l.fno, os.Stdout)
			} else {
				l.out = io.MultiWriter(l.fno)
			}
		}
	}
	l.rollingFile()
	if level >= l.logLevel {
		s := fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), level, msg)
		if level >= 40 && l.logLevel >= 20 {
			println(s)
		}
		if l.logClassified {
			s = l.cWorker.EncryptNoTail(s)
		}
		l.chanWriteLog <- s
		// go func() {
		// 	defer func() { recover() }()
		// 	if l.logClassified {
		// 		s = l.cWorker.EncryptNoTail(s)
		// 	}
		// 	fmt.Fprintln(l.out, s)
		// }()
	}
}

// Debug writelog with level 10
func (l *MxLog) Debug(msg string) {
	l.writeLog(msg, logDebug)
}

// Info writelog with level 20
func (l *MxLog) Info(msg string) {
	l.writeLog(msg, logInfo)
}

// Warning writelog with level 30
func (l *MxLog) Warning(msg string) {
	l.writeLog(msg, logWarning)
}

// Error writelog with level 40
func (l *MxLog) Error(msg string) {
	l.writeLog(msg, logError)
}

// System writelog with level 90
func (l *MxLog) System(msg string) {
	l.writeLog(msg, logSystem)
}

// DebugFormat writelog with level 10
func (l *MxLog) DebugFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), logDebug)
}

// InfoFormat writelog with level 20
func (l *MxLog) InfoFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), logInfo)
}

// WarningFormat writelog with level 30
func (l *MxLog) WarningFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), logWarning)
}

// ErrorFormat writelog with level 40
func (l *MxLog) ErrorFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), logError)
}

// SystemFormat writelog with level 90
func (l *MxLog) SystemFormat(f string, msg ...interface{}) {
	l.writeLog(fmt.Sprintf(f, msg...), logSystem)
}

// CurrentFileSize current file size
// func (l *MxLog) CurrentFileSize() int64 {
// 	return l.fileSize
// }

// EnableGZ EnableGZ
// func (l *MxLog) EnableGZ(b bool) {
// 	l.enablegz = b
// }

// InitNewLogger [Discard] use NewLogger() instead
func InitNewLogger(p string) Logger {
	return NewLogger(filepath.Dir(p), filepath.Base(p), 20, 15)
}

// NewLogger init logger
// 日志保存路径，日志文件名，日志级别，日志保留天数
func NewLogger(d, f string, logLevel, logDays int) Logger {
	switch logLevel {
	case -1:
		return &NilLogger{}
	case 0:
		return &StdLogger{}
	}
	t := time.Now()
	mylog := &MxLog{
		expired:       int64(logDays)*24*60*60 - 10,
		fileMaxSize:   maxFileSize,
		fname:         f,
		fileIndex:     0,
		fileDay:       t.Day(),
		fileHour:      t.Hour(),
		logDir:        d,
		logLevel:      logLevel,
		chanWriteLog:  make(chan string, 100),
		enablegz:      true,
		logClassified: false,
		cWorker:       GetNewCryptoWorker(CryptoAES128CBC),
	}
	mylog.cWorker.SetKey(":@9j&%D5pA!ISE_P", "JTHp^#h#<2|bgL}e")
	if IsExist(filepath.Join(GetExecDir(), ".safemode")) {
		mylog.logClassified = true
	}

	for i := byte(255); i > 0; i-- {
		if IsExist(filepath.Join(mylog.logDir, fmt.Sprintf("%s.%v.%d.log", mylog.fname, t.Format(FileTimeFormat), i))) {
			mylog.fileIndex = i
		}
	}
	mylog.newFile()

	// 创建写入线程
	go func() {
		var locker sync.WaitGroup
		locker.Add(1)
	RUN:
		go func() {
			defer func() {
				recover()
				locker.Done()
			}()
			for s := range mylog.chanWriteLog {
				fmt.Fprintln(mylog.out, s)
			}
		}()
		locker.Wait()
		goto RUN
	}()

	return mylog
}

// 检查文件大小,返回是否需要切分文件
func (l *MxLog) rolledWithFileSize() bool {
	if l.fileHour == time.Now().Hour() {
		return false
	}
	l.fileHour = time.Now().Hour()
	fs, ex := os.Stat(l.pathNow)
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
	l.nameNow = fmt.Sprintf("%s.%v.%d.log", l.fname, t.Format(FileTimeFormat), l.fileIndex)
	// 比对文件名，若不同则重新设置io
	if l.nameNow == l.nameOld {
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
	if !l.enablegz || len(s) == 0 || !IsExist(filepath.Join(l.logDir, s)) {
		return
	}
	go func(s string) {
		err := ZIPFile(l.logDir, s, true)
		if err != nil {
			l.Error("zip log file error: " + s + " " + err.Error())
			return
		}
	}(s)
}

// 清理旧日志
func (l *MxLog) clearFile() {
	// 若未设置超时，则不清理
	if l.expired == 0 {
		return
	}
	go func() {
		defer func() { recover() }()
		// 遍历文件夹
		lstfno, ex := ioutil.ReadDir(l.logDir)
		if ex != nil {
			println(fmt.Sprintf("clear log files error: %s", ex.Error()))
			return
		}
		t := time.Now()
		for _, fno := range lstfno {
			if fno.IsDir() || !strings.Contains(fno.Name(), l.fname) { // 忽略目录，不含日志名的文件，以及当前文件
				continue
			}
			// 比对文件生存期
			if t.Unix()-fno.ModTime().Unix() >= l.expired {
				os.Remove(filepath.Join(l.logDir, fno.Name()))
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
	l.nameNow = fmt.Sprintf("%s.%v.%d.log", l.fname, t.Format(FileTimeFormat), l.fileIndex)
	l.pathNow = filepath.Join(l.logDir, l.nameNow)
	// 直接写入当日日志
	if l.fname == "" {
		l.out = io.MultiWriter(os.Stdout)
	} else {
		// 打开文件
		l.fno, l.err = os.OpenFile(l.pathNow, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
		if l.err != nil {
			ioutil.WriteFile("logerr.log", []byte("Log file open error: "+l.err.Error()), 0664)
			l.out = io.MultiWriter(os.Stdout)
		} else {
			if l.logLevel <= 10 {
				l.out = io.MultiWriter(l.fno, os.Stdout)
			} else {
				l.out = io.MultiWriter(l.fno)
			}
		}
		// 判断是否压缩旧日志
		if l.enablegz {
			l.zipFile(l.nameOld)
		}
	}
	l.nameOld = l.nameNow
}
