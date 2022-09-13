package logger

import (
	"io"
	"os"
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

// DefaultWriter 返回日志Writer
func (l *NilLogger) DefaultWriter() io.Writer { return nil }

// // StdLogger 空日志
// type StdLogger struct{}

// // Debug Debug
// func (l *StdLogger) Debug(msgs string) {
// 	l.Write(toBytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
// }

// // Info Info
// func (l *StdLogger) Info(msgs string) {
// 	l.Write(toBytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
// }

// // Warning Warning
// func (l *StdLogger) Warning(msgs string) {
// 	l.Write(toBytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
// }

// // Error Error
// func (l *StdLogger) Error(msgs string) {
// 	l.Write(toBytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
// }

// // System System
// func (l *StdLogger) System(msgs string) {
// 	l.Write(toBytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msgs)))
// }

// // Writer Writer
// func (l *StdLogger) Write(p []byte) (n int, err error) {
// 	println(string(p) + "\n")
// 	return len(p), nil
// }

// // DefaultWriter 返回日志Writer
// func (l *StdLogger) DefaultWriter() io.Writer { return os.Stdout }

// StdLogger mx log
type StdLogger struct {
	out      io.Writer
	logLevel int
}

// DefaultWriter out
func (l *StdLogger) DefaultWriter() io.Writer {
	return l.out
}

// WriteLog 写日志
func (l *StdLogger) writeLog(msg string, level int) {
	if l.logLevel < 1 {
		return
	}
	// 写日志
	if l.logLevel == 1 || level >= l.logLevel {
		l.out.Write(toBytes(msg))
	}
}

// Debug writelog with level 10
func (l *StdLogger) Debug(msg string) {
	l.writeLog(msg, logDebug)
}

// Info writelog with level 20
func (l *StdLogger) Info(msg string) {
	l.writeLog(msg, logInfo)
}

// Warning writelog with level 30
func (l *StdLogger) Warning(msg string) {
	l.writeLog(msg, logWarning)
}

// Error writelog with level 40
func (l *StdLogger) Error(msg string) {
	l.writeLog(msg, logError)
}

// System writelog with level 90
func (l *StdLogger) System(msg string) {
	l.writeLog(msg, logSystem)
}

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
	opt := &OptLog{
		FileDir:  d,
		Filename: f,
		AutoRoll: logLevel >= 10,
		MaxDays:  logDays,
		ZipFile:  logDays > 7,
	}
	if logLevel == 10 {
		return &StdLogger{
			logLevel: logLevel,
			out:      io.MultiWriter(NewWriter(opt), os.Stdout),
		}
	}
	return &StdLogger{
		logLevel: logLevel,
		out:      NewWriter(opt),
	}
}

// NewConsoleLogger 返回一个纯控制台日志输出器
func NewConsoleLogger() Logger {
	opt := &OptLog{
		FileDir:  "",
		Filename: "",
		AutoRoll: false,
	}
	return &StdLogger{
		logLevel: 1,
		out:      NewWriter(opt),
	}
}
