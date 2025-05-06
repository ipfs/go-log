package log

import (
	"encoding/json"
	"io"
	"testing"
)

func TestLogLevel(t *testing.T) {
	const subsystem = "log-level-test"
	logger := Logger(subsystem)
	done := make(chan struct{})
	readLog := func(reader *PipeReader) {
		defer func() {
			done <- struct{}{}
		}()
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
	}
	reader := NewPipeReader()
	go readLog(reader)
	logger.Debugw("foo")
	_ = logger.Sync()
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
	<-done

	if err := SetLogLevel(subsystem, "debug"); err != nil {
		t.Error(err)
	}
	reader = NewPipeReader()
	go readLog(reader)
	logger.Debugw("bar")
	_ = logger.Sync()
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
	<-done

	SetAllLoggers(LevelInfo)
	reader = NewPipeReader()
	go readLog(reader)
	logger.Debugw("baz")
	// ignore the error because
	// https://github.com/uber-go/zap/issues/880
	_ = logger.Sync()
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
	<-done
}
