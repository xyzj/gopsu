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
	maxFileSize      = 209715200 // 200M
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
	chanWrite    chan logMessage
	chanClose    chan bool
	useChan      bool
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
	l.fileLogger.SetFlags(logFlagsHigh)
	l.conLogger.SetFlags(logFlagsHigh)
}

func (l *MxLog) setlogFlagsLow() {
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
	l.fileLogger = log.New(fno, "", logFlagsLow)
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
	l.fileMaxSize = c
}

// SetLogLevel set file & console log level
func (l *MxLog) SetLogLevel(loglevel byte, conlevel byte) {
	l.logLevel = loglevel
	l.conLevel = conlevel
}

// StartWriteLog StartWriteLog
func (l *MxLog) StartWriteLog() {
	l.useChan = true
	go func() {
		closeme := false
		for {
			if closeme {
				break
			}
			select {
			case msg := <-l.chanWrite:
				l.fileCheckNoLock()
				if msg.level >= l.logLevel {
					l.fileLogger.Println(msg.msg)
					l.fileSize += int64(len(msg.msg) + 17)
				}
				if msg.level >= l.conLevel {
					l.conLogger.Println(msg.msg)
				}
			case <-l.chanClose:
				closeme = true
			}
		}
	}()
}

func (l *MxLog) writeLog(msg string, level byte) {
	l.fileCheck()
	if level >= l.logLevel {
		l.fileLogger.Println(msg)
		l.fileSize += int64(len(msg) + 17)
	}
	if level >= l.conLevel {
		l.conLogger.Println(msg)
	}
}

// Debug writelog with level 10
func (l *MxLog) Debug(msg string) {
	if l.useChan {
		l.chanWrite <- logMessage{
			msg:   msg,
			level: 10,
		}
	} else {
		l.writeLog(msg, logDebug)
	}
}

// Info writelog with level 20
func (l *MxLog) Info(msg string) {
	if l.useChan {
		l.chanWrite <- logMessage{
			msg:   msg,
			level: 20,
		}
	} else {
		l.writeLog(msg, logInfo)
	}
}

// Warning writelog with level 30
func (l *MxLog) Warning(msg string) {
	if l.useChan {
		l.chanWrite <- logMessage{
			msg:   msg,
			level: 30,
		}
	} else {
		l.writeLog(msg, logWarning)
	}
}

// Error writelog with level 40
func (l *MxLog) Error(msg string) {
	if l.useChan {
		l.chanWrite <- logMessage{
			msg:   msg,
			level: 40,
		}
	} else {
		l.writeLog(msg, logError)
	}
	// _, fn, lno, _ := runtime.Caller(1)
	// go l.writeLog(fmt.Sprintf("[%s:%d] %s", filepath.Base(fn), lno, msg), logError)
}

// System writelog with level 90
func (l *MxLog) System(msg string) {
	if l.useChan {
		l.chanWrite <- logMessage{
			msg:   msg,
			level: 90,
		}
	} else {
		l.writeLog(msg, logSystem)
	}
}

// CurrentFileSize current file size
func (l *MxLog) CurrentFileSize() int64 {
	return l.fileSize
}

func (l *MxLog) SetChannelSize(i int) {
	l.chanWrite = make(chan logMessage, i)
}

// Close close logger
func (l *MxLog) Close() error {
	// defer l.logFile.Close()
	// l.fileLogger = nil
	// l.conLogger = nil
	if l.useChan {
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
		fileLogger:   log.New(fno, "", logFlagsTimeOnly),
		fileSize:     0,
		fileName:     filepath.Base(f),
		fileDir:      filepath.Dir(f),
		logLevel:     logDebug,
		logDate:      Time2Stamp(fmt.Sprintf("%d-%02d-%02d 00:00:00", t.Year(), t.Month(), t.Day())),
		// logDate:     t.Unix(),
		conLevel:    logWarning,
		conLogger:   log.New(os.Stdout, "", logFlagsTimeOnly),
		indexNumber: 0,
		mu:          new(sync.Mutex),
		chanWrite:   make(chan logMessage, 500),
		chanClose:   make(chan bool),
		useChan:     false,
	}
	mylog.getFileSize()
	return mylog
}
