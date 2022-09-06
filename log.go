package gopsu

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/xyzj/gopsu/logger"
)

const (
	logDebug            = 10
	logInfo             = 20
	logWarning          = 30
	logError            = 40
	logSystem           = 90
	logformater         = "%s %s"
	logformaterWithName = "%s [%s] %s"
)

// Logger 日志接口
type Logger interface {
	Debug(msgs string)
	Info(msgs string)
	Warning(msgs string)
	Error(msgs string)
	System(msgs string)
	// DebugFormat(f string, msgs ...interface{})
	// InfoFormat(f string, msgs ...interface{})
	// WarningFormat(f string, msgs ...interface{})
	// ErrorFormat(f string, msgs ...interface{})
	// SystemFormat(f string, msgs ...interface{})
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

// // DebugFormat Debug
// func (l *NilLogger) DebugFormat(f string, msg ...interface{}) {}

// // InfoFormat Info
// func (l *NilLogger) InfoFormat(f string, msg ...interface{}) {}

// // WarningFormat Warning
// func (l *NilLogger) WarningFormat(f string, msg ...interface{}) {}

// // ErrorFormat Error
// func (l *NilLogger) ErrorFormat(f string, msg ...interface{}) {}

// // SystemFormat System
// func (l *NilLogger) SystemFormat(f string, msg ...interface{}) {}

// DefaultWriter 返回日志Writer
func (l *NilLogger) DefaultWriter() io.Writer { return nil }

// StdLogger 空日志
type StdLogger struct{}

// Debug Debug
func (l *StdLogger) Debug(msgs string) {
	l.Write(Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
}

// Info Info
func (l *StdLogger) Info(msgs string) {
	l.Write(Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
}

// Warning Warning
func (l *StdLogger) Warning(msgs string) {
	l.Write(Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
}

// Error Error
func (l *StdLogger) Error(msgs string) {
	l.Write(Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
}

// System System
func (l *StdLogger) System(msgs string) {
	l.Write(Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
}

// Writer Writer
func (l *StdLogger) Write(p []byte) (n int, err error) {
	println(String(p) + "\n")
	return len(p), nil
}

// // DebugFormat Debug
// func (l *StdLogger) DebugFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), 10)
// }

// // InfoFormat Info
// func (l *StdLogger) InfoFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), 20)
// }

// // WarningFormat Warning
// func (l *StdLogger) WarningFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), 30)
// }

// // ErrorFormat Error
// func (l *StdLogger) ErrorFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), 40)
// }

// // SystemFormat System
// func (l *StdLogger) SystemFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), 90)
// }

// DefaultWriter 返回日志Writer
func (l *StdLogger) DefaultWriter() io.Writer { return os.Stdout }

// MxLog mx log
type MxLog struct {
	out      io.Writer
	logLevel int
}

// DefaultWriter out
func (l *MxLog) DefaultWriter() io.Writer {
	return l.out
}

// WriteLog 写日志
func (l *MxLog) writeLog(msg string, level int) {
	if l.logLevel < 1 {
		return
	}
	msg = fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msg)
	// 写日志
	if level >= l.logLevel {
		l.out.Write(Bytes(msg))

		if level >= 40 && l.logLevel > 1 {
			println(msg)
		}
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
// func (l *MxLog) DebugFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logDebug)
// }

// // InfoFormat writelog with level 20
// func (l *MxLog) InfoFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logInfo)
// }

// // WarningFormat writelog with level 30
// func (l *MxLog) WarningFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logWarning)
// }

// // ErrorFormat writelog with level 40
// func (l *MxLog) ErrorFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logError)
// }

// // SystemFormat writelog with level 90
// func (l *MxLog) SystemFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logSystem)
// }

// NewLogger init logger
//
// d: 日志保存路径
//
// f:日志文件名，为空时仅输出到控制台
//
// logLevel 日志等级，1-输出到控制台，10-debug（输出到控制台和文件）,20-info（输出到文件）,30-warning（输出到文件）,40-error（输出到控制台和文件）,90-system（输出到控制台和文件）
//
// logDays 日志文件保留天数
func NewLogger(d, f string, logLevel, logDays int) Logger {
	if logLevel < 10 {
		f = ""
	}
	if f == "" {
		logLevel = 1
	}
	opt := &logger.OptLog{
		FileDir:  d,
		Filename: f,
		AutoRoll: logLevel >= 10,
		MaxDays:  logDays,
		ZipFile:  logDays > 7,
	}
	return &MxLog{
		logLevel: logLevel,
		out:      logger.NewWriter(opt),
	}
}
