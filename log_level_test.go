package log

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
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
			if err != nil {
				require.ErrorIs(t, err, io.EOF)
				return
			}

			require.Equal(t, "bar", entry.Message)
			require.NotEmpty(t, entry.Caller, "no caller in log entry")
		}
	}()
	logger.Debugw("foo")
	require.True(t, logger.LevelEnabled(LevelError))
	require.False(t, logger.LevelEnabled(LevelDebug))
	err := SetLogLevel(subsystem, "debug")
	require.NoError(t, err)
	require.True(t, logger.LevelEnabled(LevelError))
	require.True(t, logger.LevelEnabled(LevelDebug))

	logger.Debugw("bar")
	SetAllLoggers(LevelInfo)
	logger.Debugw("baz")
	// ignore the error because
	// https://github.com/uber-go/zap/issues/880
	_ = logger.Sync()
	err = reader.Close()
	require.NoError(t, err)
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
		require.Equal(t, expected, DefaultLevel(), "wrong default level")

		// empty string subsystem
		lvl, err := SubsystemLevelName("")
		require.NoError(t, err)
		require.Equal(t, expected.String(), lvl)
	}
}

func TestGetSubsystemLevelNames(t *testing.T) {
	originalConfig := GetConfig()
	defer SetupLogging(originalConfig)

	// Clear any state from previous tests first
	clearLoggerState()

	SetupLogging(Config{Level: LevelWarn, Stderr: true})
	base := SubsystemLevelNames()

	require.Len(t, base, 1, "SubsystemLevelNames returned map with wrong size")
	require.Equal(t, LevelWarn, DefaultLevel())
	require.Equal(t, LevelWarn.String(), base[DefaultName], "SubsystemLevelNames has wrong default")

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

	require.Equal(t, DefaultLevel().String(), all[DefaultName], "SubsystemLevelNames has wrong default")
	for name, want := range expected {
		got, ok := all[name]
		require.True(t, ok, "missing key %q in SubsystemLevelNames", name)
		require.Equal(t, want.String(), got)
	}

	// dynamic logger test
	const dynKey = "dynamic"
	_ = Logger(dynKey)
	err := SetLogLevel(dynKey, "fatal")
	require.NoError(t, err)

	all = SubsystemLevelNames()
	lvl, ok := all[dynKey]
	require.True(t, ok, "missing %q key after creation", dynKey)
	require.Equalf(t, LevelFatal.String(), lvl, "wrong value for key %q", dynKey)

	// ensure immutability
	const newKey = "newkey"
	snapshot := SubsystemLevelNames()
	snapshot[DefaultName] = DefaultLevel().String()
	snapshot[newKey] = LevelInfo.String()

	// ensure original state unchanged
	fresh := SubsystemLevelNames()
	require.Equal(t, LevelError.String(), fresh[DefaultName], "immutable check failed, wrong default level")
	require.NotContains(t, fresh, newKey, "new key should not leak into internal map")
}

func TestLogLevelString(t *testing.T) {
	testLevels := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}
	expectNames := []string{"debug", "info", "warn", "error"}

	for i := range testLevels {
		require.Equal(t, expectNames[i], testLevels[i].String(), "unexpected name for level")
	}
}
