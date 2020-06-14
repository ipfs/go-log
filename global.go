package log

import "sync"

var predefinedGlobalLogger = Logger("global")
var globalMu sync.Mutex

func Debug(args ...interface{}) {
	predefinedGlobalLogger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	predefinedGlobalLogger.Debugf(format, args...)
}

func Error(args ...interface{}) {
	predefinedGlobalLogger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	predefinedGlobalLogger.Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	predefinedGlobalLogger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	predefinedGlobalLogger.Fatalf(format, args...)
}

func Info(args ...interface{}) {
	predefinedGlobalLogger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	predefinedGlobalLogger.Infof(format, args...)
}

func Panic(args ...interface{}) {
	predefinedGlobalLogger.Panic(args...)
}

func Panicf(format string, args ...interface{}) {
	predefinedGlobalLogger.Panicf(format, args...)
}

func Warn(args ...interface{}) {
	predefinedGlobalLogger.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	predefinedGlobalLogger.Warnf(format, args...)
}

func Warning(args ...interface{}) {
	predefinedGlobalLogger.Warn(args...)
}

func Warningf(format string, args ...interface{}) {
	predefinedGlobalLogger.Warnf(format, args...)
}

func ReplaceGlobalLogger(logger *ZapEventLogger) (undo func()) {
	globalMu.Lock()
	undo = func() {
		ReplaceGlobalLogger(predefinedGlobalLogger)
	}
	predefinedGlobalLogger = logger
	globalMu.Unlock()

	return undo
}
