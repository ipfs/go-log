package log

import (
	"encoding/json"
	"io"
	"testing"
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
		} else if lvl != LevelName(expected) {
			t.Errorf("GetLogLevel() = %v, want %v", lvl, LevelName(expected))
		}

		// explicit "*"
		lvl, err = GetLogLevel("*")
		if err != nil {
			t.Errorf(`GetLogLevel("*") returned error: %v`, err)
		} else if lvl != LevelName(expected) {
			t.Errorf(`GetLogLevel("*") = %v, want %v`, lvl, LevelName(expected))
		}

		// empty string
		lvl, err = GetLogLevel("")
		if err != nil {
			t.Errorf(`GetLogLevel("") returned error: %v`, err)
		} else if lvl != LevelName(expected) {
			t.Errorf(`GetLogLevel("") = %v, want %v`, lvl, LevelName(expected))
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
	if base["*"] != LevelName(LevelWarn) {
		t.Errorf("baseline GetAllLogLevels()[\"*\"] = %v; want %v", base["*"], LevelName(LevelWarn))
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

	if all["*"] != LevelName(LevelError) {
		t.Errorf(`GetAllLogLevels()["*"] = %v; want %v`, all["*"], LevelName(LevelError))
	}
	for name, want := range expected {
		got, ok := all[name]
		if !ok {
			t.Errorf("missing key %q in GetAllLogLevels()", name)
			continue
		}
		if got != LevelName(want) {
			t.Errorf(`GetAllLogLevels()["%s"] = %v; want %v`, name, got, LevelName(want))
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
	} else if lvl != LevelName(LevelFatal) {
		t.Errorf(`GetAllLogLevels()["dynamic"] = %v; want %v`, lvl, LevelName(LevelFatal))
	}

	// ensure immutability
	snapshot := GetAllLogLevels()
	snapshot["*"] = LevelName(LevelDebug)
	snapshot["newkey"] = LevelName(LevelInfo)

	// ensure original state unchanged
	fresh := GetAllLogLevels()
	if fresh["*"] != LevelName(LevelError) {
		t.Errorf(`immutable check failed: fresh["*"] = %v; want %v`, fresh["*"], LevelName(LevelError))
	}
	if _, exists := fresh["newkey"]; exists {
		t.Error(`immutable check failed: "newkey" should not leak into real map`)
	}
}

func TestLevelName(t *testing.T) {
	testLevels := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}
	expectNames := []string{"debug", "info", "warn", "error"}

	for i := range testLevels {
		name := LevelName(testLevels[i])
		if name != expectNames[i] {
			t.Errorf("unexpected name for level: expected %s, got %s", expectNames[i], name)
		}
	}
}
