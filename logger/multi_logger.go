package logger

import (
	"io"
)

type MultiLogger struct {
	outs []*StdLogger
}

func NewMultiLogger(writers ...*StdLogger) Logger {
	return &MultiLogger{
		outs: writers,
	}
}

// DefaultWriter out
func (l *MultiLogger) DefaultWriter() io.Writer {
	if len(l.outs) > 0 {
		return l.outs[0].Out
	}
	return nil
}

// WriteLog 写日志
func (l *MultiLogger) writeLog(msg string, level LogLevel) {
	// 写日志
	for _, o := range l.outs {
		if level >= o.LogLevel && o.Out != nil {
			o.Out.Write(toBytes(msg))
		}
	}
}

// Debug writelog with level 10
func (l *MultiLogger) Debug(msg string) {
	l.writeLog("[10] "+msg, LogDebug)
}

// Info writelog with level 20
func (l *MultiLogger) Info(msg string) {
	l.writeLog("[20] "+msg, LogInfo)
}

// Warning writelog with level 30
func (l *MultiLogger) Warning(msg string) {
	l.writeLog("[30] "+msg, LogWarning)
}

// Error writelog with level 40
func (l *MultiLogger) Error(msg string) {
	l.writeLog("[40] "+msg, LogError)
}

// System writelog with level 90
func (l *MultiLogger) System(msg string) {
	l.writeLog("[90] "+msg, LogSystem)
}
