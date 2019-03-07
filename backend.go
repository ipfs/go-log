package log

import (
	"errors"
	"io"
	"sync"

	logrus "github.com/sirupsen/logrus"
)

var ErrNoSuchLogger = errors.New("Error: No such logger")

// loggers is the set of loggers in the system
var loggerMutex sync.RWMutex
var loggers = map[string]*logrus.Entry{}

// SetAllLoggers changes the logging.Level of all loggers to lvl
func SetAllLogLevel(lvl Level) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	for _, lg := range loggers {
		lg.Logger.SetLevel(lvl)
	}
}

// SetLogLevel changes the log level of a specific subsystem
func SetLogLevel(name string, level string) error {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}

	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	// Check if we have a logger by that name
	if log, ok := loggers[name]; !ok {
		return ErrNoSuchLogger
	} else {
		log.Logger.SetLevel(lvl)
	}
	return nil
}

func SetAllLogFormat(lgfmt Formatter) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	for _, lg := range loggers {
		lg.Logger.SetFormatter(lgfmt)
	}
}

func SetLogFormat(name string, lgfmt Formatter) error {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	if log, ok := loggers[name]; !ok {
		return ErrNoSuchLogger
	} else {
		log.Logger.SetFormatter(lgfmt)
	}
	return nil
}

func SetAllLogFormatJSON() {
	SetAllLogFormat(DefaultJSONFormat)
}

func SetAllLogFormatText() {
	SetAllLogFormat(DefaultTextFormat)
}

func SetAllLogOutput(out io.Writer) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	for _, lg := range loggers {
		lg.Logger.SetOutput(out)
	}
}

func SetLogOutput(name string, out io.Writer) error {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	if log, ok := loggers[name]; !ok {
		return ErrNoSuchLogger
	} else {
		log.Logger.SetOutput(out)
	}
	return nil
}

// GetSubsystems returns a slice containing the names of the current loggers
func GetSubsystems() []string {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	subs := make([]string, 0, len(loggers))

	for k := range loggers {
		subs = append(subs, k)
	}
	return subs
}

func getOrCreateLogger(name string) *logrus.Entry {
	log := loggers[name]
	if log == nil {
		// create a new logrus logger
		log = logrus.New().WithField("system", name)
		// report the calling method and filename by default
		log.Logger.SetReportCaller(true)
		log.Logger.SetFormatter(DefaultTextFormat)
		loggers[name] = log
		return log
	}

	return log
}
