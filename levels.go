package log

import "go.uber.org/zap/zapcore"

// LogLevel represents a log severity level. Use the package variables as an
// enum.
type LogLevel zapcore.Level

var (
	LevelDebug  = LogLevel(zapcore.DebugLevel)
	LevelInfo   = LogLevel(zapcore.InfoLevel)
	LevelWarn   = LogLevel(zapcore.WarnLevel)
	LevelError  = LogLevel(zapcore.ErrorLevel)
	LevelDPanic = LogLevel(zapcore.DPanicLevel)
	LevelPanic  = LogLevel(zapcore.PanicLevel)
	LevelFatal  = LogLevel(zapcore.FatalLevel)
)

// LevelFromString parses a string-based level and returns the corresponding
// LogLevel.
//
// Supported strings are: DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL, and
// their lower-case forms.
//
// The returned LogLevel must be discarded if error is not nil.
func LevelFromString(level string) (LogLevel, error) {
	lvl := zapcore.InfoLevel // zero value
	err := lvl.Set(level)
	return LogLevel(lvl), err
}

// GetLogLevel returns the current log level for a given subsystem as a string.
// Passing name="*" or name="" returns the defaultLevel.
func GetLogLevel(name string) (string, error) {
	if name == "*" || name == "" {
		loggerMutex.RLock()
		defLvl := defaultLevel
		loggerMutex.RUnlock()
		return zapcore.Level(defLvl).String(), nil
	}
	if lvl, ok := levels[name]; ok {
		return zapcore.Level(LogLevel(lvl.Level())).String(), nil
	}
	return "", ErrNoSuchLogger
}

// GetAllLogLevels returns a map of all current log levels for all subsystems as strings.
// The map includes a special "*" key that represents the defaultLevel.
func GetAllLogLevels() map[string]string {
	result := make(map[string]string, len(levels)+1)

	// Add the default level with "*" key
	loggerMutex.RLock()
	defLvl := defaultLevel
	loggerMutex.RUnlock()
	result["*"] = zapcore.Level(defLvl).String()

	// Add all subsystem levels
	for name, level := range levels {
		result[name] = zapcore.Level(LogLevel(level.Level())).String()
	}

	return result
}
