package log

import "go.uber.org/zap/zapcore"

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

func LevelFromString(level string) (LogLevel, error) {
	lvl := zapcore.InfoLevel // zero value
	err := lvl.Set(level)
	return LogLevel(lvl), err
}
