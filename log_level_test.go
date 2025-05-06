package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestLogLevel(t *testing.T) {
	const subsystem = "log-level-test"
	logger := Logger(subsystem)
	readLog := func(reader *PipeReader, errCh chan error) {
		decoder := json.NewDecoder(reader)
		var entry struct {
			Message string `json:"msg"`
			Caller  string `json:"caller"`
		}
		err := decoder.Decode(&entry)
		switch err {
		default:
			errCh <- err
			return
		case io.EOF:
			errCh <- nil
			return
		case nil:
		}
		if entry.Message != "bar" {
			errCh <- fmt.Errorf("unexpected message: %s", entry.Message)
			return
		}
		if entry.Caller == "" {
			errCh <- errors.New("no caller in log entry")
			return
		}
		errCh <- nil
	}

	err := SetLogLevel(subsystem, "error")
	if err != nil {
		t.Error(err)
	}
	reader := NewPipeReader()
	errCh := make(chan error, 1)
	go readLog(reader, errCh)
	logger.Debug("foo")
	_ = logger.Sync()
	if err = reader.Close(); err != nil {
		t.Error(err)
	}
	if err = <-errCh; err != nil {
		t.Error(err)
	}

	if err = SetLogLevel(subsystem, "debug"); err != nil {
		t.Error(err)
	}
	reader = NewPipeReader()
	go readLog(reader, errCh)
	logger.Debug("bar")
	_ = logger.Sync()
	if err = reader.Close(); err != nil {
		t.Error(err)
	}
	if err = <-errCh; err != nil {
		t.Error(err)
	}

	SetAllLoggers(LevelInfo)
	reader = NewPipeReader()
	go readLog(reader, errCh)
	logger.Debug("baz")
	// ignore the error because
	// https://github.com/uber-go/zap/issues/880
	_ = logger.Sync()
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
	if err = <-errCh; err != nil {
		t.Error(err)
	}
}
