package mxgo

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	logDebug         = 10
	logInfo          = 20
	logWarning       = 30
	logError         = 40
	logSystem        = 90
	logFlagsHigh     = log.Ldate | log.Ltime | log.Lshortfile
	logFlagsLow      = log.Ldate | log.Ltime
	logFlagsTimeOnly = log.Lmicroseconds
	maxFileCount     = 30
	maxFileSize      = 1048576000 // 1G
)

var (
	asyncCache = 1000
	logFlags   = logFlagsTimeOnly
)

// MxLog mx log
type MxLog struct {
	fileFullPath string
	fileSize     int64
	fileMaxCount uint16
	fileMaxSize  int64
	fileLogger   *log.Logger
	fileName     string
	fileDir      string
	logFile      *os.File
	logLevel     byte
	logDate      int64
	conLevel     byte
	conLogger    *log.Logger
	indexNumber  byte
	mu           *sync.Mutex
	chanWrite    chan *logMessage
	chanClose    chan bool
	writeAsync   bool
	asyncLock    sync.WaitGroup
	chanWatcher  chan string
}

type logMessage struct {
	msg   string
	level byte
}

type sortFiles struct {
	modTime  int64
	fullPath string
}

type fileSorter []sortFiles

func (fs fileSorter) Len() int {
	return len(fs)
}

func (fs fileSorter) Less(i, j int) bool {
	return fs[i].modTime > fs[j].modTime
}

func (fs fileSorter) Swap(i, j int) {
	fs[i], fs[j] = fs[j], fs[i]
}

func (l *MxLog) clearLogFile() {
	lstfno, ex := ioutil.ReadDir(l.fileDir)
	if ex != nil {
		println(fmt.Sprintf("--- clear log files error: %s", ex.Error()))
		// l.Error(ex.Error())
	}
	sf := make(fileSorter, 0)
	for _, fno := range lstfno {
		if fno.IsDir() || !strings.Contains(fno.Name(), l.fileName) { // 忽略目录
			continue
		}
		sf = append(sf, sortFiles{fno.ModTime().UnixNano(), filepath.Join(l.fileDir, fno.Name())})
	}
	sort.Sort(sf)
	for k, v := range sf {
		if uint16(k) > l.fileMaxCount {
			os.Remove(v.fullPath)
		}
	}
}

func (l *MxLog) setlogFlagsHigh() {
	logFlags = logFlagsHigh
	l.fileLogger.SetFlags(logFlagsHigh)
	l.conLogger.SetFlags(logFlagsHigh)
}

func (l *MxLog) setlogFlagsLow() {
	logFlags = logFlagsLow
	l.fileLogger.SetFlags(logFlagsLow)
	l.conLogger.SetFlags(logFlagsLow)
}

func (l *MxLog) splitLogFile(t *time.Time) {
	ex := l.logFile.Close()
	if ex != nil {
		l.conLogger.Println(ex)
	}
	l.fileLogger = nil
	l.logFile = nil
	var nf string
	for true {
		if l.indexNumber == 0 {
			nf = filepath.Join(l.fileDir, fmt.Sprintf("%s.%d-%02d-%02d", l.fileName, t.Year(), t.Month(), t.Day()))
		} else {
			nf = filepath.Join(l.fileDir, fmt.Sprintf("%s.%d-%02d-%02d.%d", l.fileName, t.Year(), t.Month(), t.Day(), l.indexNumber))
		}
		l.indexNumber++
		a := IsExist(nf)
		if a == false {
			break
		}
	}
	os.Rename(l.fileFullPath, nf)
	l.indexNumber = 0
	fno, ex := os.OpenFile(l.fileFullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if ex != nil {
		fmt.Println(ex)
	}
	l.logFile = fno
	l.fileLogger = log.New(fno, "", logFlags)
	l.getFileSize()
	l.clearLogFile()
}

func (l *MxLog) fileCheck() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.fileCheckNoLock()
}

func (l *MxLog) fileCheckNoLock() {
	t := time.Now()
	// 检查文件是否存在,不存在重新创建
	a := IsExist(l.fileFullPath)
	if a == false {
		l.splitLogFile(&t)
		return
	}
	nts := Time2Stamp(fmt.Sprintf("%d-%02d-%02d 00:00:00", t.Year(), t.Month(), t.Day()))
	// 按日期重命名
	if l.logDate != nts {
		l.logDate = nts
		d, _ := time.ParseDuration("-24h")
		ot := t.Add(d)
		l.splitLogFile(&ot)
		return
	}
	// 按大小重命名
	if l.fileSize >= l.fileMaxSize {
		l.splitLogFile(&t)
		return
	}
}

func (l *MxLog) getFileSize() {
	f, ex := os.Stat(l.fileFullPath)
	if ex != nil {
		l.fileSize = 0
	}
	l.fileSize = f.Size()
}

// SetMaxFileCount set max log file count
func (l *MxLog) SetMaxFileCount(c uint16) {
	l.fileMaxCount = c
}

// SetMaxFileSize set max log file size in M
func (l *MxLog) SetMaxFileSize(c int64) {
	l.fileMaxSize = c * 1048576
}

// SetLogLevel set file & console log level
func (l *MxLog) SetLogLevel(loglevel byte, conlevel byte) {
	l.logLevel = loglevel
	l.conLevel = conlevel
}

// SetAsync 设置异步写入参数
func (l *MxLog) SetAsync(c int) {
	if c <= 0 {
		l.chanClose <- true
		l.writeAsync = false
	}
	if c < 1000 {
		c = 1000
	}
	if c > 10000 {
		c = 10000
	}
	l.chanWrite = make(chan *logMessage, c)
	l.writeAsync = true

	go l.coreWatcher()
	go l.writeLogAsync()

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
			case "close":
				closeme = true
				break
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
				l.chanWatcher <- "mxlog"
			} else {
				l.chanWatcher <- "close"
			}
		}()
		closeme := false
		for {
			if closeme {
				break
			}
			select {
			case msg := <-l.chanWrite:
				l.writeLog(msg.msg, msg.level, false)
			case <-l.chanClose:
				if !closeme {
					closeme = true
					close(l.chanWrite)
					if len(l.chanWrite) > 0 {
						for {
							msg, ok := <-l.chanWrite
							if !ok {
								break
							}
							l.writeLog(msg.msg, msg.level, false)
						}
					}
				}
			}
		}
	}()
}

func (l *MxLog) writeLog(msg string, level byte, lock bool) {
	if lock {
		l.fileCheck()
	} else {
		l.fileCheckNoLock()
	}
	if level >= l.logLevel {
		l.fileLogger.Println(msg)
		l.fileSize += int64(len(msg) + 17)
	}

	if level >= l.conLevel {
		switch level {
		case logDebug:
			l.conLogger.Println(GreenText(msg))
		case logInfo:
			l.conLogger.Println(msg)
		case logWarning:
			l.conLogger.Println(YellowText(msg))
		case logError:
			l.conLogger.Println(RedText(msg))
		case logSystem:
			l.conLogger.Println(ColorText(FColorCyan, BColorBlack, TextHighlight, msg))
		default:
			l.conLogger.Println(msg)
		}
	}
}

// Debug writelog with level 10
func (l *MxLog) Debug(msg string) {
	msg = fmt.Sprintf("[10] %s", msg)
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
	msg = fmt.Sprintf("[20] %s", msg)
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
	msg = fmt.Sprintf("[30] %s", msg)
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
	msg = fmt.Sprintf("[40] %s", msg)
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
	msg = fmt.Sprintf("[90] %s", msg)
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
	// defer l.logFile.Close()
	// l.fileLogger = nil
	// l.conLogger = nil
	if l.writeAsync {
		l.chanClose <- true
	}
	return l.logFile.Close()
}

// InitNewLogger init logger
func InitNewLogger(f string) *MxLog {
	fno, ex := os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if ex != nil {
		fmt.Println(ex)
	}
	t := time.Now()
	mylog := &MxLog{
		logFile:      fno,
		fileFullPath: f,
		fileMaxCount: maxFileCount,
		fileMaxSize:  maxFileSize,
		fileLogger:   log.New(fno, "", logFlags),
		fileSize:     0,
		fileName:     filepath.Base(f),
		fileDir:      filepath.Dir(f),
		logLevel:     logDebug,
		logDate:      Time2Stamp(fmt.Sprintf("%d-%02d-%02d 00:00:00", t.Year(), t.Month(), t.Day())),
		// logDate:     t.Unix(),
		conLevel:    logWarning,
		conLogger:   log.New(os.Stdout, "", logFlags),
		indexNumber: 0,
		mu:          new(sync.Mutex),
		chanWrite:   make(chan *logMessage, 1000),
		chanClose:   make(chan bool, 2),
		chanWatcher: make(chan string, 2),
		writeAsync:  false,
	}
	mylog.getFileSize()
	return mylog
}
