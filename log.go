// Package log is the logging library used by IPFS
// (https://github.com/ipfs/go-ipfs). It uses a modified version of
// https://godoc.org/github.com/whyrusleeping/go-logging .
package log

import (
//"io"
)

var log = Logger("golog")

// StandardLogger provides API compatibility with standard printf loggers
// eg. go-logging
type StandardLogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

type ExtendedLogger interface {
	StandardLogger

	WithField(key string, value interface{}) *Entry
	WithFields(fileds Fields) *Entry
	WithError(err error) *Entry

	//SetOutput(out io.Writer)
	//SetLevel(lvl Level)
	//SetFormatter(lgmft Formatter)
}

// Logger retrieves an event logger by name
func Logger(system string) ExtendedLogger {
	if len(system) == 0 {
		panic("Missing logger name parameter")
	}

	logger := getOrCreateLogger(system)

	return &goLogger{system: system, ExtendedLogger: logger}
}

// goLogger implements the ExtendedLogger interface
type goLogger struct {
	ExtendedLogger
	system string
}
