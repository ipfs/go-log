package log

import (
	"encoding/json"
	"io"
	"testing"
)

func TestLogLevel(t *testing.T) {
	const subsystem = "log-level-test"
	logger := Logger(subsystem)
	reader := NewPipeReader()
	done := make(chan struct{})
	go func() {
		defer close(done)
		decoder := json.NewDecoder(reader)
		for {
			var entry struct {
				Message string `json:"msg"`
				Caller  string `json:"caller"`
			}
			err := decoder.Decode(&entry)
			switch err {
			default:
				t.Error(err)
				return
			case io.EOF:
				return
			case nil:
			}
			if entry.Message != "bar" {
				t.Errorf("unexpected message: %s", entry.Message)
			}
			if entry.Caller == "" {
				t.Errorf("no caller in log entry")
			}
		}
	}()
	logger.Debugw("foo")
	if err := SetLogLevel(subsystem, "debug"); err != nil {
		t.Error(err)
	}
	logger.Debugw("bar")
	SetAllLoggers(LevelInfo)
	logger.Debugw("baz")
	// ignore the error because
	// https://github.com/uber-go/zap/issues/880
	_ = logger.Sync()
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
	<-done
}
