package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
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

	t.Run("disabled by default", func(t *testing.T) {
		beforeSetup := slog.Default()

		cfg := Config{
			Format: PlaintextOutput,
			Level:  LevelInfo,
			Stderr: true,
		}

		SetupLogging(cfg)

		// slog.Default should NOT have changed automatically
		if slog.Default() != beforeSetup {
			t.Error("slog.Default() should NOT be automatically set by SetupLogging")
		}

		// Test that slog logs still work via SlogHandler()
		slog.Info("test message", "key", "value")
	})

	t.Run("enabled via GOLOG_CAPTURE_DEFAULT_SLOG=true", func(t *testing.T) {
		beforeSetup := slog.Default()

		// Set env var to enable
		os.Setenv("GOLOG_CAPTURE_DEFAULT_SLOG", "true")
		defer os.Unsetenv("GOLOG_CAPTURE_DEFAULT_SLOG")

		cfg := Config{
			Format: PlaintextOutput,
			Level:  LevelInfo,
			Stderr: true,
		}

		SetupLogging(cfg)

		// slog.Default should have changed
		if slog.Default() == beforeSetup {
			t.Error("slog.Default() should be modified when GOLOG_CAPTURE_DEFAULT_SLOG=true")
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
	// Save and restore global state
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

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

func TestSlogGroupConversion(t *testing.T) {
	var buf bytes.Buffer
	ws := zapcore.AddSync(&buf)
	testCore := newCore(JSONOutput, ws, LevelDebug)
	logger := slog.New(newZapToSlogBridge(testCore))

	decode := func(t *testing.T) map[string]any {
		t.Helper()
		var m map[string]any
		if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
			t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
		}
		return m
	}

	t.Run("nested groups with mixed types", func(t *testing.T) {
		buf.Reset()
		logger.Info("msg", slog.Group("outer",
			slog.String("s", "hello"),
			slog.Int("n", 42),
			slog.Bool("b", true),
			slog.Group("inner", slog.String("k", "v")),
		))
		got := decode(t)
		outer, ok := got["outer"].(map[string]any)
		if !ok {
			t.Fatalf("expected outer to be an object, got %T: %v", got["outer"], got["outer"])
		}
		if outer["s"] != "hello" || outer["n"] != float64(42) || outer["b"] != true {
			t.Errorf("unexpected scalar contents: %v", outer)
		}
		inner, ok := outer["inner"].(map[string]any)
		if !ok {
			t.Fatalf("expected inner to be an object, got %T", outer["inner"])
		}
		if inner["k"] != "v" {
			t.Errorf("unexpected inner contents: %v", inner)
		}
	})

	t.Run("fxevent-style synthetic-array group", func(t *testing.T) {
		// fxevent.slogStrings packs []string as a Group with numeric-string keys.
		// Without group support this rendered as [{"Key":"0","Value":{}},...].
		buf.Reset()
		logger.Info("msg", slog.Group("stacktrace",
			slog.String("0", "frame0"),
			slog.String("1", "frame1"),
		))
		got := decode(t)
		st, ok := got["stacktrace"].(map[string]any)
		if !ok {
			t.Fatalf("expected stacktrace to be an object, got %T: %v", got["stacktrace"], got["stacktrace"])
		}
		if st["0"] != "frame0" || st["1"] != "frame1" {
			t.Errorf("unexpected stacktrace contents: %v", st)
		}
	})

	t.Run("empty-key group is inlined", func(t *testing.T) {
		buf.Reset()
		logger.Info("msg", slog.Group("wrap",
			slog.Group("", slog.String("inlined", "yes")),
		))
		got := decode(t)
		wrap, ok := got["wrap"].(map[string]any)
		if !ok {
			t.Fatalf("expected wrap to be an object, got %T", got["wrap"])
		}
		if wrap["inlined"] != "yes" {
			t.Errorf("expected inlined attr at parent level, got: %v", wrap)
		}
	})

	t.Run("empty group is skipped", func(t *testing.T) {
		buf.Reset()
		logger.Info("msg", slog.Group("ignored"), slog.String("kept", "v"))
		got := decode(t)
		if _, present := got["ignored"]; present {
			t.Errorf("empty group should be omitted, got: %v", got)
		}
		if got["kept"] != "v" {
			t.Errorf("expected sibling attr to survive, got: %v", got)
		}
	})

	t.Run("group with time, duration, float, uint", func(t *testing.T) {
		buf.Reset()
		when := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
		logger.Info("msg", slog.Group("g",
			slog.Time("t", when),
			slog.Duration("d", 1500*time.Millisecond),
			slog.Float64("f", 2.5),
			slog.Uint64("u", uint64(1)<<33),
		))
		got := decode(t)
		g, ok := got["g"].(map[string]any)
		if !ok {
			t.Fatalf("expected g to be an object, got %T: %v", got["g"], got["g"])
		}
		if g["t"] != "2025-01-02T03:04:05.000Z" {
			t.Errorf("unexpected time: %v", g["t"])
		}
		if g["d"] != 1.5 {
			t.Errorf("unexpected duration: %v", g["d"])
		}
		if g["f"] != 2.5 {
			t.Errorf("unexpected float: %v", g["f"])
		}
		if g["u"] != float64(uint64(1)<<33) {
			t.Errorf("unexpected uint: %v", g["u"])
		}
	})

	t.Run("top-level empty-key group inlines into record", func(t *testing.T) {
		buf.Reset()
		logger.Info("msg", slog.Group("",
			slog.String("inlined", "yes"),
			slog.Int("n", 7),
		))
		got := decode(t)
		if got["inlined"] != "yes" {
			t.Errorf("expected inlined string at record level: %v", got)
		}
		if got["n"] != float64(7) {
			t.Errorf("expected inlined int at record level: %v", got)
		}
		if _, present := got[""]; present {
			t.Errorf("empty-key group should not produce an empty-key field, got: %v", got)
		}
	})
}

func TestSubsystemAwareLevelControl(t *testing.T) {
	// Save and restore global state
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	loggerMutex.Lock()
	originalLevels := levels
	originalDefaultLevel := defaultLevel
	levels = make(map[string]zap.AtomicLevel)
	defaultLevel = LevelError
	loggerMutex.Unlock()
	defer func() {
		loggerMutex.Lock()
		levels = originalLevels
		defaultLevel = originalDefaultLevel
		loggerMutex.Unlock()
	}()

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
	// Save and restore global state
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	loggerMutex.Lock()
	originalLevels := levels
	originalDefaultLevel := defaultLevel
	levels = make(map[string]zap.AtomicLevel)
	defaultLevel = LevelError
	loggerMutex.Unlock()
	defer func() {
		loggerMutex.Lock()
		levels = originalLevels
		defaultLevel = originalDefaultLevel
		loggerMutex.Unlock()
	}()

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

	// Explicitly set go-log's handler as slog.Default
	slog.SetDefault(slog.New(SlogHandler()))

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

// createGologshimStyleLogger simulates gologshim.Logger() behavior.
// After SetupLogging is called, slog.Default() contains go-log's bridge.
// This function checks for the bridge and uses it with WithAttrs to add the subsystem.
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

	// Explicitly set go-log's handler as slog.Default
	slog.SetDefault(slog.New(SlogHandler()))

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
