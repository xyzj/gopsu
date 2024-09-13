package logger

import (
	"io"
	"strings"
	"unsafe"
)

type LogLevel byte

const (
	LogDebug            LogLevel = 10
	LogInfo             LogLevel = 20
	LogWarning          LogLevel = 30
	LogError            LogLevel = 40
	LogSystem           LogLevel = 90
	logformater                  = "%s %s"
	logformaterWithName          = "%s [%s] %s"
)

var lineend = []byte{10}

// Logger 日志接口
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warning(msg string)
	Error(msg string)
	System(msg string)
	// DebugFormat(f string, msg ...interface{})
	// InfoFormat(f string, msg ...interface{})
	// WarningFormat(f string, msg ...interface{})
	// ErrorFormat(f string, msg ...interface{})
	DefaultWriter() io.Writer
}

// NilLogger 空日志
type NilLogger struct{}

// Debug Debug
func (l *NilLogger) Debug(msg string) {}

// Info Info
func (l *NilLogger) Info(msg string) {}

// Warning Warning
func (l *NilLogger) Warning(msg string) {}

// Error Error
func (l *NilLogger) Error(msg string) {}

// // DebugFormat Debug
// func (l *NilLogger) DebugFormat(f string, msg ...interface{}) {}

// // InfoFormat Info
// func (l *NilLogger) InfoFormat(f string, msg ...interface{}) {}

// // WarningFormat Warning
// func (l *NilLogger) WarningFormat(f string, msg ...interface{}) {}

// // ErrorFormat Error
// func (l *NilLogger) ErrorFormat(f string, msg ...interface{}) {}

// System System
func (l *NilLogger) System(msg string) {}

// DefaultWriter 返回日志Writer
func (l *NilLogger) DefaultWriter() io.Writer { return nil }

// // StdLogger 空日志
// type StdLogger struct{}

// // Debug Debug
// func (l *StdLogger) Debug(msg string) {
// 	l.Write(tools.Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msg)))
// }

// // Info Info
// func (l *StdLogger) Info(msg string) {
// 	l.Write(tools.Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msg)))
// }

// // Warning Warning
// func (l *StdLogger) Warning(msg string) {
// 	l.Write(tools.Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msg)))
// }

// // Error Error
// func (l *StdLogger) Error(msg string) {
// 	l.Write(tools.Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msg)))
// }

// // System System
// func (l *StdLogger) System(msg string) {
// 	l.Write(tools.Bytes(fmt.Sprintf(logformater, time.Now().Format(ShortTimeFormat), msg)))
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
	cout     io.Writer
	out      io.Writer
	clevel   map[LogLevel]struct{}
	logLevel LogLevel
}

// DefaultWriter out
func (l *StdLogger) DefaultWriter() io.Writer {
	return l.out
}

// WriteLog 写日志
func (l *StdLogger) writeLog(msg string, level LogLevel) {
	// 写日志
	if level >= l.logLevel && l.out != nil {
		l.out.Write(toBytes(msg))
	}
	// 写控制台
	if _, ok := l.clevel[level]; ok && l.cout != nil {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		l.cout.Write(toBytes(msg))
	}
}

// Debug writelog with level 10
func (l *StdLogger) Debug(msg string) {
	l.writeLog("[10] "+msg, LogDebug)
}

// Info writelog with level 20
func (l *StdLogger) Info(msg string) {
	l.writeLog("[20] "+msg, LogInfo)
}

// Warning writelog with level 30
func (l *StdLogger) Warning(msg string) {
	l.writeLog("[30] "+msg, LogWarning)
}

// Error writelog with level 40
func (l *StdLogger) Error(msg string) {
	l.writeLog("[40] "+msg, LogError)
}

// System writelog with level 90
func (l *StdLogger) System(msg string) {
	l.writeLog("[90] "+msg, LogSystem)
}

// // DebugFormat writelog with level 10
// func (l *StdLogger) DebugFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logDebug)
// }

// // InfoFormat writelog with level 20
// func (l *StdLogger) InfoFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logInfo)
// }

// // WarningFormat writelog with level 30
// func (l *StdLogger) WarningFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logWarning)
// }

// // ErrorFormat writelog with level 40
// func (l *StdLogger) ErrorFormat(f string, msg ...interface{}) {
// 	l.writeLog(fmt.Sprintf(f, msg...), logError)
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
//
// delayWrite 是否延迟写入，在日志密集时，可减少磁盘io，但可能导致日志丢失
func NewLogger(opt *OptLog) Logger {
	if opt.Filename == "" {
		return NewConsoleLogger()
	}
	// func NewLogger(d, f string, logLevel, logDays int, delayWrite bool, consoleLevels ...byte) Logger {
	// var ll byte
	// switch logLevel {
	// case 10, 20, 30, 40, 90:
	// 	ll = byte(logLevel)
	// default:
	// 	ll = 10
	// }
	// if f == "" {
	// 	return NewConsoleLogger()
	// }
	// opt := &OptLog{
	// 	FileDir:    d,
	// 	Filename:   f,
	// 	AutoRoll:   ll >= 10,
	// 	MaxDays:    logDays,
	// 	ZipFile:    logDays > 7,
	// 	DelayWrite: delayWrite,
	// }
	cl := make(map[LogLevel]struct{})
	for _, v := range opt.ConsoleLevels {
		cl[v] = struct{}{}
	}
	return &StdLogger{
		logLevel: opt.FileLevel,
		out:      NewWriter(opt),
		cout:     NewConsoleWriter(),
		clevel:   cl,
	}
}

// NewConsoleLogger 返回一个纯控制台日志输出器
func NewConsoleLogger() Logger {
	return &StdLogger{
		// out:      NewWriter(nil),
		// logLevel: 10,
		cout: NewConsoleWriter(),
		clevel: map[LogLevel]struct{}{
			10: {},
			20: {},
			30: {},
			40: {},
			90: {},
		},
	}
}

func toBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			cap int
		}{s, len(s)},
	))
}
