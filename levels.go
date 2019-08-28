package log

import "go.uber.org/zap/zapcore"

func stringToZap(level string) (zapcore.Level, error) {
	lvl := zapcore.DebugLevel
	err := lvl.Set(level)
	return lvl, err
}
