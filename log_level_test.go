package log

import (
	"encoding/json"
	"io"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestLogLevel(t *testing.T) {
	const subsystem = "log-level-test"

	// Save original config and restore after test
	originalConfig := GetConfig()
	defer SetupLogging(originalConfig)

	// Reset to a known state with error level default
	SetupLogging(Config{
		Level:  LevelError,
		Stderr: true,
	})

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

// Helper function to clear logger state between tests
func clearLoggerState() {
	clear(loggers)
	clear(levels)
}

func TestGetDefaultLevel(t *testing.T) {
	originalConfig := GetConfig()
	defer SetupLogging(originalConfig)

	// Clear any state from previous tests first
	clearLoggerState()

	testCases := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}

	for _, expected := range testCases {
		SetupLogging(Config{Level: expected, Stderr: true})

		// empty string arg
		lvl, err := GetLogLevel("")
		if err != nil {
			t.Errorf("GetLogLevel() returned error: %v", err)
		} else if lvl != zapcore.Level(expected).String() {
			t.Errorf("GetLogLevel() = %v, want %v", lvl, zapcore.Level(expected).String())
		}

		// explicit "*"
		lvl, err = GetLogLevel("*")
		if err != nil {
			t.Errorf(`GetLogLevel("*") returned error: %v`, err)
		} else if lvl != zapcore.Level(expected).String() {
			t.Errorf(`GetLogLevel("*") = %v, want %v`, lvl, zapcore.Level(expected).String())
		}

		// empty string
		lvl, err = GetLogLevel("")
		if err != nil {
			t.Errorf(`GetLogLevel("") returned error: %v`, err)
		} else if lvl != zapcore.Level(expected).String() {
			t.Errorf(`GetLogLevel("") = %v, want %v`, lvl, zapcore.Level(expected).String())
		}
	}
}

func TestGetAllLogLevels(t *testing.T) {
	originalConfig := GetConfig()
	defer SetupLogging(originalConfig)

	// Clear any state from previous tests first
	clearLoggerState()

	SetupLogging(Config{Level: LevelWarn, Stderr: true})
	base := GetAllLogLevels()

	if len(base) != 1 {
		t.Errorf("baseline GetAllLogLevels() length = %d; want 1", len(base))
	}
	if base["*"] != zapcore.Level(LevelWarn).String() {
		t.Errorf("baseline GetAllLogLevels()[\"*\"] = %v; want %v", base["*"], zapcore.Level(LevelWarn).String())
	}

	expected := map[string]LogLevel{
		"test1": LevelDebug,
		"test2": LevelInfo,
		"test3": LevelWarn,
	}
	SetupLogging(Config{
		Level:           LevelError,
		SubsystemLevels: expected,
		Stderr:          true,
	})

	all := GetAllLogLevels()

	if all["*"] != zapcore.Level(LevelError).String() {
		t.Errorf(`GetAllLogLevels()["*"] = %v; want %v`, all["*"], zapcore.Level(LevelError).String())
	}
	for name, want := range expected {
		got, ok := all[name]
		if !ok {
			t.Errorf("missing key %q in GetAllLogLevels()", name)
			continue
		}
		if got != zapcore.Level(want).String() {
			t.Errorf(`GetAllLogLevels()["%s"] = %v; want %v`, name, got, zapcore.Level(want).String())
		}
	}

	// dynamic logger test
	_ = Logger("dynamic")
	if err := SetLogLevel("dynamic", "fatal"); err != nil {
		t.Fatalf("SetLogLevel(dynamic) failed: %v", err)
	}

	all = GetAllLogLevels()
	if lvl, ok := all["dynamic"]; !ok {
		t.Error(`missing "dynamic" key after creation`)
	} else if lvl != zapcore.Level(LevelFatal).String() {
		t.Errorf(`GetAllLogLevels()["dynamic"] = %v; want %v`, lvl, zapcore.Level(LevelFatal).String())
	}

	// ensure immutability
	snapshot := GetAllLogLevels()
	snapshot["*"] = zapcore.Level(LevelDebug).String()
	snapshot["newkey"] = zapcore.Level(LevelInfo).String()

	// ensure original state unchanged
	fresh := GetAllLogLevels()
	if fresh["*"] != zapcore.Level(LevelError).String() {
		t.Errorf(`immutable check failed: fresh["*"] = %v; want %v`, fresh["*"], zapcore.Level(LevelError).String())
	}
	if _, exists := fresh["newkey"]; exists {
		t.Error(`immutable check failed: "newkey" should not leak into real map`)
	}
}
