package gopsu

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	maxFileLife = 30 * 24 * 60 * 60
	maxFileSize = 1048576000 // 1G
)

var (
	asyncCache = 1000
)

// MxLog mx log
type MxLog struct {
	fileFullPath  string
	fileSize      int64
	fileMaxLife   int64
	fileMaxSize   int64
	fileLogger    *log.Logger
	fileName      string
	fileNameNow   string
	fileNameOld   string
	fileIndex     byte
	fileDir       string
	fileDay       int
	fno           *os.File
	logLevel      int
	conLogger     *log.Logger
	enablegz      bool
	err           error
	fileLock      sync.RWMutex
	chanWrite     chan *logMessage
	chanClose     chan bool
	writeAsync    bool
	asyncLock     sync.WaitGroup
	chanWatcher   chan string
	DefaultWriter io.Writer
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
	l.fileMaxLife = c * 24 * 60 * 60
}

// SetMaxFileCount [Discard] use SetMaxFileLife() instead
func (l *MxLog) SetMaxFileCount(c uint16) {
	l.SetMaxFileLife(int64(c))
}

// SetMaxFileSize set max log file size in M
func (l *MxLog) SetMaxFileSize(c int64) {
	l.fileMaxSize = c * 1048576
}

// SetLogLevel set file & console log level
func (l *MxLog) SetLogLevel(loglevel int, conlevel ...int) {
	l.logLevel = loglevel

	if l.logLevel <= 10 {
		l.DefaultWriter = io.MultiWriter(l.fno, os.Stdout)
	} else {
		l.DefaultWriter = io.MultiWriter(l.fno)
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
		t := time.NewTicker(time.Hour)
		for {
			if closeme {
				break
			}
			select {
			case msg := <-l.chanWrite:
				l.writeLog(msg.msg, msg.level)
			case <-t.C:
				fs, ex := os.Stat(l.fileFullPath)
				if ex == nil {
					if fs.Size() > 1048576000 {
						if l.fileIndex >= 255 {
							l.fileIndex = 0
						} else {
							l.fileIndex++
						}
					}
				}
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
		fmt.Fprintln(l.DefaultWriter, fmt.Sprintf("%s [%02d] %s", time.Now().Format(ShortTimeFormat), level, msg))
		// l.fileLogger.Println(msg)
		// l.fileSize += int64(len(msg) + 17)
	}
}

// Debug writelog with level 10
func (l *MxLog) Debug(msg string) {
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
func (l *MxLog) Info(msg string) {
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
func (l *MxLog) Warning(msg string) {
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
func (l *MxLog) Error(msg string) {
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
func (l *MxLog) System(msg string) {
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
	// defer l.fno.Close()
	// l.fileLogger = nil
	// l.conLogger = nil
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
		fileDir:     d,
		logLevel:    logDebug,
		chanWrite:   make(chan *logMessage, 5000),
		chanClose:   make(chan bool, 2),
		chanWatcher: make(chan string, 2),
		writeAsync:  false,
		enablegz:    true,
	}

	for i := byte(0); i < 255; i++ {
		if IsExist(filepath.Join(mylog.fileDir, fmt.Sprintf("%s.%v.%d.log", mylog.fileName, t.Format(FileTimeFromat), i))) {
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

func (l *MxLog) rollingFileNoLock() bool {
	t := time.Now()
	// if l.fileSize > l.fileMaxSize {
	// 	l.fileIndex++
	// }
	l.fileNameNow = fmt.Sprintf("%s.%v.%d.log", l.fileName, t.Format(FileTimeFromat), l.fileIndex)
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
		zfile := filepath.Join(l.fileDir, s+".zip")
		ofile := filepath.Join(l.fileDir, s)

		newZipFile, err := os.Create(zfile)
		if err != nil {
			return
		}
		defer newZipFile.Close()

		zipWriter := zip.NewWriter(newZipFile)
		defer zipWriter.Close()

		zipfile, err := os.Open(ofile)
		if err != nil {
			return
		}
		defer zipfile.Close()
		info, err := zipfile.Stat()
		if err != nil {
			return
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return
		}
		header.Name = s
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return
		}
		if _, err = io.Copy(writer, zipfile); err != nil {
			return
		}
		// 删除已压缩的旧日志
		os.Remove(filepath.Join(l.fileDir, s))
	}()
}

// 清理旧日志
func (l *MxLog) clearFile() {
	// 若未设置超时，则不清理
	if l.fileMaxLife == 0 {
		return
	}
	// 遍历文件夹
	lstfno, ex := ioutil.ReadDir(l.fileDir)
	if ex != nil {
		println(fmt.Sprintf("clear log files error: %s", ex.Error()))
	}
	t := time.Now()
	for _, fno := range lstfno {
		if fno.IsDir() || !strings.Contains(fno.Name(), l.fileName) || strings.Contains(fno.Name(), ".current") { // 忽略目录，不含日志名的文件，以及当前文件
			continue
		}
		// 比对文件生存期
		if t.Unix()-fno.ModTime().Unix() > l.fileMaxLife {
			os.Remove(filepath.Join(l.fileDir, fno.Name()))
		}
	}
}

// 创建新日志文件
func (l *MxLog) newFile() {
	// 使用文件链接创建当前日志文件
	// 文件不存在时创建
	// if !IsExist(l.fileFullName) {
	// 	f, err := os.Create(l.fileFullName)
	// 	if err == nil {
	// 		l.Close()
	// 	}
	// }
	// 删除旧的文件链接
	// os.Remove(l.fileFullName)
	// // 创建当前日志链接
	// l.err = os.Symlink(l.fileName, l.fileFullName)
	// if l.err != nil {
	// 	println("Symlink log file error: " + l.err.Error())
	// }
	if l.fileDay != time.Now().Day() {
		l.fileIndex = 0
	}
	l.fileNameNow = fmt.Sprintf("%s.%v.%d.log", l.fileName, time.Now().Format(FileTimeFromat), l.fileIndex)
	l.fileFullPath = filepath.Join(l.fileDir, l.fileNameNow)
	// 直接写入当日日志
	// 打开文件
	l.fno, l.err = os.OpenFile(l.fileFullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if l.err != nil {
		println("Log file open error: " + l.err.Error())
	}
	if l.logLevel <= 10 {
		l.DefaultWriter = io.MultiWriter(l.fno, os.Stdout)
	} else {
		l.DefaultWriter = io.MultiWriter(l.fno)
	}
	// l.fileLogger = log.New(l.fno, "", log.Lmicroseconds)
	// l.fileSize = l.getFileSize()

	// 判断是否压缩旧日志
	if l.enablegz {
		l.zipFile(l.fileNameOld)
	}
	l.fileNameOld = l.fileNameNow
}
