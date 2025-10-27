package log

import (
	"bufio"
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

// goLogBridge is used to detect go-log's slog bridge
type goLogBridge interface {
	GoLogBridge()
}

func TestSlogInterop(t *testing.T) {
	// Save initial state
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	t.Run("enabled by default", func(t *testing.T) {
		beforeSetup := slog.Default()

		cfg := Config{
			Format: PlaintextOutput,
			Level:  LevelInfo,
			Stderr: true,
		}

		SetupLogging(cfg)

		// slog.Default should have changed automatically
		if slog.Default() == beforeSetup {
			t.Error("slog.Default() should be automatically set by SetupLogging")
		}

		// Test that slog logs work
		slog.Info("test message", "key", "value")
	})

	t.Run("disabled via GOLOG_CAPTURE_DEFAULT_SLOG=false", func(t *testing.T) {
		beforeSetup := slog.Default()

		// Set env var to disable
		os.Setenv("GOLOG_CAPTURE_DEFAULT_SLOG", "false")
		defer os.Unsetenv("GOLOG_CAPTURE_DEFAULT_SLOG")

		cfg := Config{
			Format: PlaintextOutput,
			Level:  LevelInfo,
			Stderr: true,
		}

		SetupLogging(cfg)

		// slog.Default should NOT have changed
		if slog.Default() != beforeSetup {
			t.Error("slog.Default() should not be modified when GOLOG_CAPTURE_DEFAULT_SLOG=false")
		}
	})
}

func TestSlogBridgeLevels(t *testing.T) {
	tests := []struct {
		slogLevel slog.Level
		wantZap   string
	}{
		{slog.LevelDebug, "debug"},
		{slog.LevelInfo, "info"},
		{slog.LevelWarn, "warn"},
		{slog.LevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(strings.ToUpper(tt.wantZap), func(t *testing.T) {
			zapLevel := slogLevelToZap(tt.slogLevel)
			if !strings.Contains(zapLevel.String(), tt.wantZap) {
				t.Errorf("slogLevelToZap(%v) = %v, want to contain %v", tt.slogLevel, zapLevel, tt.wantZap)
			}
		})
	}
}

func TestSlogAttrFieldConversions(t *testing.T) {
	// Test that slog.Attr types are correctly converted to zap fields
	var buf bytes.Buffer
	ws := zapcore.AddSync(&buf)
	testCore := newCore(PlaintextOutput, ws, LevelDebug)
	setPrimaryCore(testCore)

	bridge := newZapToSlogBridge(testCore)
	slog.SetDefault(slog.New(bridge))

	// Test all slog attribute types
	logger := slog.Default()

	// String
	logger.Info("test", "string_key", "string_value")
	output := buf.String()
	if !strings.Contains(output, "string_key") || !strings.Contains(output, "string_value") {
		t.Error("String attribute not correctly converted")
	}

	// Int64
	buf.Reset()
	logger.Info("test", "int64_key", int64(42))
	output = buf.String()
	if !strings.Contains(output, "int64_key") || !strings.Contains(output, "42") {
		t.Error("Int64 attribute not correctly converted")
	}

	// Uint64
	buf.Reset()
	logger.Info("test", "uint64_key", uint64(100))
	output = buf.String()
	if !strings.Contains(output, "uint64_key") || !strings.Contains(output, "100") {
		t.Error("Uint64 attribute not correctly converted")
	}

	// Float64 - this tests the bug fix
	buf.Reset()
	logger.Info("test", "float64_key", float64(3.14159))
	output = buf.String()
	if !strings.Contains(output, "float64_key") || !strings.Contains(output, "3.14159") {
		t.Errorf("Float64 attribute not correctly converted: %s", output)
	}

	// Bool
	buf.Reset()
	logger.Info("test", "bool_key", true)
	output = buf.String()
	if !strings.Contains(output, "bool_key") || !strings.Contains(output, "true") {
		t.Error("Bool attribute not correctly converted")
	}

	// Duration
	buf.Reset()
	logger.Info("test", "duration_key", slog.DurationValue(5*time.Second))
	output = buf.String()
	if !strings.Contains(output, "duration_key") || !strings.Contains(output, "5") {
		t.Error("Duration attribute not correctly converted")
	}

	// Time
	buf.Reset()
	testTime := time.Date(2025, 10, 26, 12, 0, 0, 0, time.UTC)
	logger.Info("test", "time_key", slog.TimeValue(testTime))
	output = buf.String()
	if !strings.Contains(output, "time_key") || !strings.Contains(output, "2025") {
		t.Error("Time attribute not correctly converted")
	}

	// Any (complex type)
	buf.Reset()
	logger.Info("test", "any_key", slog.AnyValue(map[string]int{"count": 5}))
	output = buf.String()
	if !strings.Contains(output, "any_key") {
		t.Error("Any attribute not correctly converted")
	}
}

func TestSubsystemAwareLevelControl(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	// Create a custom core that writes to our buffer
	ws := zapcore.AddSync(&buf)
	testCore := newCore(PlaintextOutput, ws, LevelDebug)
	setPrimaryCore(testCore)
	setAllLoggers(LevelError)

	// Set up slog bridge
	bridge := newZapToSlogBridge(testCore)
	slog.SetDefault(slog.New(bridge))

	// Create a subsystem-aware logger (mimics gologshim behavior)
	logger := slog.Default().With("logger", "test-subsystem")

	// Try to log debug message - should be filtered at ERROR level
	logger.Debug("this should not appear")
	output := buf.String()
	if strings.Contains(output, "this should not appear") {
		t.Error("Debug log should be filtered when subsystem is at ERROR level")
	}

	// Change level dynamically using SetLogLevel (mimics `ipfs log level` RPC)
	err := SetLogLevel("test-subsystem", "debug")
	if err != nil {
		t.Fatalf("SetLogLevel failed: %v", err)
	}

	// Now log debug message - should appear
	buf.Reset()
	logger.Debug("this should appear", "key", "value")
	output = buf.String()

	if !strings.Contains(output, "this should appear") {
		t.Error("Debug log should appear after SetLogLevel to debug")
	}
	if !strings.Contains(output, "test-subsystem") {
		t.Error("Log should contain subsystem name")
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Error("Log should contain key-value attributes")
	}

	// Change back to info - debug should be filtered again
	err = SetLogLevel("test-subsystem", "info")
	if err != nil {
		t.Fatalf("SetLogLevel failed: %v", err)
	}

	buf.Reset()
	logger.Debug("this should not appear again")
	output = buf.String()
	if strings.Contains(output, "this should not appear again") {
		t.Error("Debug log should be filtered when level changed back to INFO")
	}

	// Info should still work
	logger.Info("info message")
	output = buf.String()
	if !strings.Contains(output, "info message") {
		t.Error("Info log should appear at INFO level")
	}
}

func TestSetLogLevelWithSlog(t *testing.T) {
	// Setup go-log
	var buf bytes.Buffer
	ws := zapcore.AddSync(&buf)
	testCore := newCore(PlaintextOutput, ws, LevelDebug)
	setPrimaryCore(testCore)
	setAllLoggers(LevelError)

	bridge := newZapToSlogBridge(testCore)
	slog.SetDefault(slog.New(bridge))

	// Create slog logger for subsystem that doesn't exist yet
	logger := slog.Default().With("logger", "test-slog-subsystem")

	// Debug should be filtered (default is error)
	logger.Debug("should not appear")
	if strings.Contains(buf.String(), "should not appear") {
		t.Error("Debug log should be filtered at ERROR level")
	}

	// Set level dynamically via SetLogLevel
	err := SetLogLevel("test-slog-subsystem", "debug")
	if err != nil {
		t.Fatalf("SetLogLevel failed: %v", err)
	}

	// Debug should now appear
	buf.Reset()
	logger.Debug("should appear")
	if !strings.Contains(buf.String(), "should appear") {
		t.Error("Debug log should appear after SetLogLevel to debug")
	}
}

func TestPipeReaderCapturesSlogLogs(t *testing.T) {
	// Setup go-log
	SetupLogging(Config{
		Format: JSONOutput,
		Level:  LevelError,
		Stderr: false,
		Stdout: false,
	})

	// Set levels for both subsystems
	SetLogLevel("test-native", "debug")
	SetLogLevel("test-slog", "debug")

	// Create pipe reader
	pipeReader := NewPipeReader()
	defer pipeReader.Close()

	// Start reading from pipe
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			buf.WriteString(scanner.Text())
			buf.WriteString("\n")
		}
		close(done)
	}()

	// Log from native go-log logger
	nativeLogger := Logger("test-native")
	nativeLogger.Debug("native debug message")

	// Log from slog (simulating gologshim behavior)
	slogLogger := slog.Default().With("logger", "test-slog")
	slogLogger.Debug("slog debug message")

	// Give logs time to flush
	time.Sleep(200 * time.Millisecond)
	pipeReader.Close()
	<-done

	output := buf.String()
	t.Logf("Pipe reader captured:\n%s", output)

	// Check both logs were captured
	if !strings.Contains(output, "test-native") {
		t.Error("Native log not captured")
	}
	if !strings.Contains(output, "native debug message") {
		t.Error("Native log message not found")
	}
	if !strings.Contains(output, "test-slog") {
		t.Error("Slog logger name not captured")
	}
	if !strings.Contains(output, "slog debug message") {
		t.Error("Slog log message not found")
	}
}

// createGologshimStyleLogger simulates gologshim.Logger() behavior
func createGologshimStyleLogger(system string) *slog.Logger {
	if _, ok := slog.Default().Handler().(goLogBridge); ok {
		attrs := []slog.Attr{slog.String("logger", system)}
		h := slog.Default().Handler().WithAttrs(attrs)
		return slog.New(h)
	}
	panic("go-log bridge not detected")
}

func TestPipeReaderCapturesGologshimStyleLogs(t *testing.T) {
	// Setup go-log
	SetupLogging(Config{
		Format: JSONOutput,
		Level:  LevelError,
		Stderr: false,
		Stdout: false,
	})

	// Verify slog.Default() has the bridge
	if _, ok := slog.Default().Handler().(goLogBridge); !ok {
		t.Fatal("slog.Default() does not have go-log bridge")
	}

	// Set levels
	SetLogLevel("test-native", "debug")
	SetLogLevel("test-gologshim", "debug")

	// Create pipe reader
	pipeReader := NewPipeReader()
	defer pipeReader.Close()

	// Start reading from pipe
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			buf.WriteString(scanner.Text())
			buf.WriteString("\n")
		}
		close(done)
	}()

	// Log from native go-log logger
	nativeLogger := Logger("test-native")
	nativeLogger.Debug("native debug message")

	// Log from gologshim-style logger
	gologshimLogger := createGologshimStyleLogger("test-gologshim")
	gologshimLogger.Debug("gologshim debug message")

	// Give logs time to flush
	time.Sleep(200 * time.Millisecond)
	pipeReader.Close()
	<-done

	output := buf.String()
	t.Logf("Pipe reader captured:\n%s", output)

	// Check both logs were captured
	if !strings.Contains(output, "test-native") {
		t.Error("Native log not captured")
	}
	if !strings.Contains(output, "native debug message") {
		t.Error("Native log message not found")
	}
	if !strings.Contains(output, "test-gologshim") {
		t.Error("Gologshim logger name not captured")
	}
	if !strings.Contains(output, "gologshim debug message") {
		t.Error("Gologshim log message not found")
	}
}
