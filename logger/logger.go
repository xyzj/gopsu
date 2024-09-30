package logger

import (
	"io"
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

// System System
func (l *NilLogger) System(msg string) {}

// DefaultWriter 返回日志Writer
func (l *NilLogger) DefaultWriter() io.Writer { return nil }

// StdLogger mx log
type StdLogger struct {
	Out      io.Writer
	LogLevel LogLevel
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
//
// delayWrite 是否延迟写入，在日志密集时，可减少磁盘io，但可能导致日志丢失
func NewLogger(opt *OptLog) Logger {
	if opt.Filename == "" {
		return NewConsoleLogger()
	}
	return &MultiLogger{
		outs: []*StdLogger{
			{
				LogLevel: opt.FileLevel,
				Out:      NewWriter(opt),
			},
		},
	}
}

// NewConsoleLogger 返回一个纯控制台日志输出器
func NewConsoleLogger() Logger {
	return &MultiLogger{
		outs: []*StdLogger{
			{
				LogLevel: LogDebug,
				Out:      NewConsoleWriter(),
			},
		},
	}
}
