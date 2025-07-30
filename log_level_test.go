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

func TestDefaultLevel(t *testing.T) {
	originalConfig := GetConfig()
	defer SetupLogging(originalConfig)

	// Clear any state from previous tests first
	clearLoggerState()

	testCases := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}

	for _, expected := range testCases {
		SetupLogging(Config{Level: expected, Stderr: true})

		// check default
		if DefaultLevel() != expected {
			t.Fatalf("wrong default level, want %v, got %v", DefaultLevel(), expected)
		}

		// empty string subsystem
		lvl, err := SubsystemLevelName("")
		if err != nil {
			t.Errorf("SubsystemLevelName returned error: %v", err)
		} else if lvl != expected.String() {
			t.Errorf("SubsystemLevelName returned %v, want %v", lvl, expected)
		}
	}
}

func TestGetSubsystemLevelNames(t *testing.T) {
	originalConfig := GetConfig()
	defer SetupLogging(originalConfig)

	// Clear any state from previous tests first
	clearLoggerState()

	SetupLogging(Config{Level: LevelWarn, Stderr: true})
	base := SubsystemLevelNames()

	if len(base) != 1 {
		t.Errorf("baseline SubsystemLevelNames() length = %d; want 1", len(base))
	}
	if DefaultLevel() != LevelWarn {
		t.Fatal("wrong default level")
	}
	if base[DefaultName] != LevelWarn.String() {
		t.Errorf("baseline SubsystemLevelNames()[\"\"] = %v; want %v", base["*"], LevelWarn.String())
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

	all := SubsystemLevelNames()

	if all[""] != DefaultLevel().String() {
		t.Errorf(`SubsystemLevelNames()[""] = %v; want %v`, all[""], DefaultLevel().String())
	}
	for name, want := range expected {
		got, ok := all[name]
		if !ok {
			t.Errorf("missing key %q in SubsystemLevelNames()", name)
			continue
		}
		if got != want.String() {
			t.Errorf(`SubsystemLevelNames()["%s"] = %v; want %v`, name, got, want.String())
		}
	}

	// dynamic logger test
	_ = Logger("dynamic")
	if err := SetLogLevel("dynamic", "fatal"); err != nil {
		t.Fatalf("SetLogLevel(dynamic) failed: %v", err)
	}

	all = SubsystemLevelNames()
	if lvl, ok := all["dynamic"]; !ok {
		t.Error(`missing "dynamic" key after creation`)
	} else if lvl != LevelFatal.String() {
		t.Errorf(`SubsystemLevelNames()["dynamic"] = %v; want %v`, lvl, LevelFatal.String())
	}

	// ensure immutability
	snapshot := SubsystemLevelNames()
	snapshot[DefaultName] = DefaultLevel().String()
	snapshot["newkey"] = LevelInfo.String()

	// ensure original state unchanged
	fresh := SubsystemLevelNames()
	if fresh[DefaultName] != LevelError.String() {
		t.Errorf(`immutable check failed: fresh[DefaultName] = %v; want %v`, fresh[DefaultName], LevelError.String())
	}
	if _, exists := fresh["newkey"]; exists {
		t.Error(`immutable check failed: "newkey" should not leak into real map`)
	}
}

func TestLogLevelString(t *testing.T) {
	testLevels := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}
	expectNames := []string{"debug", "info", "warn", "error"}

	for i := range testLevels {
		if testLevels[i].String() != expectNames[i] {
			t.Errorf("unexpected name for level: expected %s, got %s", expectNames[i], testLevels[i].String())
		}
	}

}
