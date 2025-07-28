package log

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGetLoggerDefault(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// Call SetupLogging again so it picks up stderr change
	SetupLogging(Config{Stderr: true})
	log := getLogger("test")

	log.Error("scooby")
	w.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}

func TestLogToFileAndStderr(t *testing.T) {
	// setup stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// setup file
	logfile, err := os.CreateTemp("", "go-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(logfile.Name())

	os.Setenv(envLoggingFile, logfile.Name())
	defer os.Unsetenv(envLoggingFile)

	// set log output env var
	os.Setenv(envLoggingOutput, "file+stderr")
	defer os.Unsetenv(envLoggingOutput)

	SetupLogging(configFromEnv())

	log := getLogger("test")

	want := "scooby"
	log.Error(want)
	w.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), want) {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

	content, err := os.ReadFile(logfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), want) {
		t.Logf("want: '%s', got: '%s'", want, string(content))
		t.Fail()
	}
}

func TestLogToFile(t *testing.T) {
	// get tmp log file
	logfile, err := os.CreateTemp("", "go-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(logfile.Name())

	// set the go-log file env var
	os.Setenv(envLoggingFile, logfile.Name())
	defer os.Unsetenv(envLoggingFile)

	SetupLogging(configFromEnv())

	log := getLogger("test")

	// write log to file
	want := "grokgrokgrok"
	log.Error(want)

	// read log file and check contents
	content, err := os.ReadFile(logfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), want) {
		t.Logf("want: '%s', got: '%s'", want, string(content))
		t.Fail()
	}
}

func TestLogLabels(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// set the go-log labels env var
	os.Setenv(envLoggingLabels, "dc=sjc-1,foobar") // foobar to ensure we don't panic on bad input.
	defer os.Unsetenv(envLoggingLabels)
	SetupLogging(configFromEnv())

	log := getLogger("test")

	log.Error("scooby")
	w.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Log(buf.String())
	if !strings.Contains(buf.String(), "{\"dc\": \"sjc-1\"}") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}

func TestSubsystemLevels(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// set the go-log labels env var
	os.Setenv(envLogging, "info,test1=debug")
	defer os.Unsetenv(envLoggingLabels)
	SetupLogging(configFromEnv())

	log1 := getLogger("test1")
	log2 := getLogger("test2")

	log1.Debug("debug1")
	log1.Info("info1")
	log2.Debug("debug2")
	log2.Info("info2")
	w.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "debug1") {
		t.Errorf("got %q, wanted it to contain debug1", buf.String())
	}
	if strings.Contains(buf.String(), "debug2") {
		t.Errorf("got %q, wanted it to not contain debug2", buf.String())
	}
	if !strings.Contains(buf.String(), "info1") {
		t.Errorf("got %q, wanted it to contain info1", buf.String())
	}
	if !strings.Contains(buf.String(), "info2") {
		t.Errorf("got %q, wanted it to contain info2", buf.String())
	}
}

func TestCustomCore(t *testing.T) {
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	// logging should work with the custom core
	SetPrimaryCore(newCore(PlaintextOutput, w1, LevelDebug))
	log := getLogger("test")
	log.Error("scooby")

	// SetPrimaryCore should replace the core in previously created loggers
	SetPrimaryCore(newCore(PlaintextOutput, w2, LevelDebug))
	log.Error("doo")

	w1.Close()
	w2.Close()

	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	if _, err := io.Copy(buf1, r1); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := io.Copy(buf2, r2); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf1.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf1.String())
	}
	if !strings.Contains(buf2.String(), "doo") {
		t.Errorf("got %q, wanted it to contain log output", buf2.String())
	}
}

func TestTeeCore(t *testing.T) {
	// configure to use a tee logger
	tee := zap.New(zapcore.NewTee(
		zap.NewNop().Core(),
		zap.NewNop().Core(),
	), zap.AddCaller())
	SetPrimaryCore(tee.Core())
	log := getLogger("test")
	log.Error("scooby")

	// replaces the tee logger with a simple one
	SetPrimaryCore(zap.NewNop().Core())
	log.Error("doo")
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

		// no args
		lvl, err := GetLogLevel()
		if err != nil {
			t.Errorf("GetLogLevel() returned error: %v", err)
		}
		if lvl != logLevelToString(expected) {
			t.Errorf("GetLogLevel() = %v, want %v", lvl, logLevelToString(expected))
		}

		// explicit "*"
		lvl, err = GetLogLevel("*")
		if err != nil {
			t.Errorf(`GetLogLevel("*") returned error: %v`, err)
		}
		if lvl != logLevelToString(expected) {
			t.Errorf(`GetLogLevel("*") = %v, want %v`, lvl, logLevelToString(expected))
		}

		// empty string
		lvl, err = GetLogLevel("")
		if err != nil {
			t.Errorf(`GetLogLevel("") returned error: %v`, err)
		}
		if lvl != logLevelToString(expected) {
			t.Errorf(`GetLogLevel("") = %v, want %v`, lvl, logLevelToString(expected))
		}

		// multi-arg test
		_ = Logger("svc")
		if err := SetLogLevel("svc", "info"); err != nil {
			t.Fatalf("SetLogLevel(svc) failed: %v", err)
		}
		// multi‑arg is ignored beyond the first
		lvl1, err := GetLogLevel("svc", "ignored")
		if err != nil {
			t.Errorf("GetLogLevel(\"svc\", \"ignored\") error: %v", err)
		}
		lvl2, err := GetLogLevel("svc")
		if err != nil {
			t.Errorf("GetLogLevel(\"svc\") error: %v", err)
		}
		if lvl1 != lvl2 {
			t.Errorf("multi‑arg mismatch: GetLogLevel(\"svc\",\"ignored\")=%v, GetLogLevel(\"svc\")=%v", lvl1, lvl2)
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
	if base["*"] != logLevelToString(LevelWarn) {
		t.Errorf("baseline GetAllLogLevels()[\"*\"] = %v; want %v", base["*"], logLevelToString(LevelWarn))
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

	if all["*"] != logLevelToString(LevelError) {
		t.Errorf(`GetAllLogLevels()["*"] = %v; want %v`, all["*"], logLevelToString(LevelError))
	}
	for name, want := range expected {
		got, ok := all[name]
		if !ok {
			t.Errorf("missing key %q in GetAllLogLevels()", name)
			continue
		}
		if got != logLevelToString(want) {
			t.Errorf(`GetAllLogLevels()["%s"] = %v; want %v`, name, got, logLevelToString(want))
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
	} else if lvl != logLevelToString(LevelFatal) {
		t.Errorf(`GetAllLogLevels()["dynamic"] = %v; want %v`, lvl, logLevelToString(LevelFatal))
	}

	// ensure immutability
	snapshot := GetAllLogLevels()
	snapshot["*"] = logLevelToString(LevelDebug)
	snapshot["newkey"] = logLevelToString(LevelInfo)

	// ensure original state unchanged
	fresh := GetAllLogLevels()
	if fresh["*"] != logLevelToString(LevelError) {
		t.Errorf(`immutable check failed: fresh["*"] = %v; want %v`, fresh["*"], logLevelToString(LevelError))
	}
	if _, exists := fresh["newkey"]; exists {
		t.Error(`immutable check failed: "newkey" should not leak into real map`)
	}
}

func TestLogToStderrAndStdout(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	stdout := os.Stdout
	os.Stderr = w
	os.Stdout = w2
	defer func() {
		os.Stderr = stderr
		os.Stdout = stdout
	}()

	os.Setenv(envLoggingOutput, "stdout+stderr")
	defer os.Unsetenv(envLoggingOutput)

	SetupLogging(configFromEnv())

	log := getLogger("test")

	want := "scooby"
	log.Error(want)
	w.Close()
	w2.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), want) {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

	buf.Reset()
	if _, err := io.Copy(buf, r2); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), want) {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}

func TestLogToStdoutOnly(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	stdout := os.Stdout
	os.Stderr = w
	os.Stdout = w2
	defer func() {
		os.Stderr = stderr
		os.Stdout = stdout
	}()

	os.Setenv(envLoggingOutput, "stdout")
	defer os.Unsetenv(envLoggingOutput)

	SetupLogging(configFromEnv())

	log := getLogger("test")

	want := "scooby"
	log.Error(want)
	w.Close()
	w2.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Should not have read anything from stderr")
	}

	buf.Reset()
	if _, err := io.Copy(buf, r2); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), want) {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}
