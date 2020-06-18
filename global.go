package log

import "sync"

var predefinedGlobalLogger = Logger("global")
var globalMu sync.RWMutex

func Debug(args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Debug(args...)
	globalMu.RUnlock()
}

func Debugf(format string, args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Debugf(format, args...)
	globalMu.RUnlock()
}

func Error(args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Error(args...)
	globalMu.RUnlock()
}

func Errorf(format string, args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Errorf(format, args...)
	globalMu.RUnlock()
}

func Fatal(args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Fatal(args...)
	globalMu.RUnlock()
}

func Fatalf(format string, args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Fatalf(format, args...)
	globalMu.RUnlock()
}

func Info(args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Info(args...)
	globalMu.RUnlock()
}

func Infof(format string, args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Infof(format, args...)
	globalMu.RUnlock()
}

func Panic(args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Panic(args...)
	globalMu.RUnlock()
}

func Panicf(format string, args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Panicf(format, args...)
	globalMu.RUnlock()
}

func Warn(args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Warn(args...)
	globalMu.RUnlock()
}

func Warnf(format string, args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Warnf(format, args...)
	globalMu.RUnlock()
}

func Warning(args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Warn(args...)
	globalMu.RUnlock()
}

func Warningf(format string, args ...interface{}) {
	globalMu.RLock()
	predefinedGlobalLogger.Warnf(format, args...)
	globalMu.RUnlock()
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
