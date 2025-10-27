package log

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGetLoggerDefault(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

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
	if _, err = io.Copy(buf, r); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}

	require.Contains(t, buf.String(), "scooby")
}

func TestLogToFileAndStderr(t *testing.T) {
	// setup stderr
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// setup file
	logfile, err := os.CreateTemp("", "go-log-test")
	require.NoError(t, err)
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
	if _, err := io.Copy(buf, r); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}

	require.Contains(t, buf.String(), want)

	content, err := os.ReadFile(logfile.Name())
	require.NoError(t, err)

	require.Contains(t, string(content), want)
}

func TestLogToFile(t *testing.T) {
	// get tmp log file
	logfile, err := os.CreateTemp("", "go-log-test")
	require.NoError(t, err)
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
	require.NoError(t, err)

	require.Contains(t, string(content), want)
}

func TestLogLabels(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

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
	if _, err = io.Copy(buf, r); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}

	t.Log(buf.String())
	require.Contains(t, buf.String(), "{\"dc\": \"sjc-1\"}")
}

func TestSubsystemLevels(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

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
	if _, err = io.Copy(buf, r); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}

	s := buf.String()
	require.Contains(t, s, "debug1")
	require.NotContains(t, s, "debug2")
	require.Contains(t, s, "info1")
	require.Contains(t, s, "info2")
}

func TestCustomCore(t *testing.T) {
	r1, w1, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")
	r2, w2, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

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
	if _, err = io.Copy(buf1, r1); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}
	if _, err = io.Copy(buf2, r2); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}

	require.Contains(t, buf1.String(), "scooby")
	require.Contains(t, buf2.String(), "doo")
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

func TestLogToStderrAndStdout(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

	r2, w2, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

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
	if _, err = io.Copy(buf, r); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}
	require.Contains(t, buf.String(), want)

	buf.Reset()
	if _, err = io.Copy(buf, r2); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}
	require.Contains(t, buf.String(), want)
}

func TestLogToStdoutOnly(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

	r2, w2, err := os.Pipe()
	require.NoError(t, err, "failed to open pipe")

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
	if _, err := io.Copy(buf, r); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}
	require.Zero(t, buf.Len())

	buf.Reset()
	if _, err := io.Copy(buf, r2); err != nil {
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}
	require.Contains(t, buf.String(), want)
}

func TestSetLogLevelAutoCreate(t *testing.T) {
	// Save and restore original state to avoid test pollution
	loggerMutex.Lock()
	originalLevels := levels
	levels = make(map[string]zap.AtomicLevel)
	loggerMutex.Unlock()
	defer func() {
		loggerMutex.Lock()
		levels = originalLevels
		loggerMutex.Unlock()
	}()

	// Set level for non-existent subsystem (should succeed)
	err := SetLogLevel("nonexistent", "debug")
	require.NoError(t, err)

	// Verify level entry was created
	loggerMutex.RLock()
	atomicLevel, exists := levels["nonexistent"]
	loggerMutex.RUnlock()

	require.True(t, exists, "level entry should be auto-created")
	require.Equal(t, zapcore.DebugLevel, atomicLevel.Level())

	// Change level (should update existing entry)
	err = SetLogLevel("nonexistent", "error")
	require.NoError(t, err)
	require.Equal(t, zapcore.ErrorLevel, atomicLevel.Level())

	// Invalid level should still fail
	err = SetLogLevel("nonexistent", "invalid")
	require.Error(t, err)
}
