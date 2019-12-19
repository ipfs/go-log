package log

import (
	"os"
	"testing"

	"github.com/sanity-io/litter"

	"github.com/google/go-cmp/cmp"
)

func clearGoLogEnvVars() {
	os.Unsetenv(envIPFSLogging)
	os.Unsetenv(envIPFSLoggingFmt)
	os.Unsetenv(envLogging)
	os.Unsetenv(envLoggingFmt)
	os.Unsetenv(envLoggingCfg)
	os.Unsetenv(envLoggingFile)
}

func TestSetupLogging(t *testing.T) {
	t.Run("default logger configuration", func(t *testing.T) {
		defer cleanup()
		defer clearGoLogEnvVars()
		clearGoLogEnvVars()
		SetupLogging()
		// zapCfg is the package global zap config variable
		got := litter.Sdump(zapCfg)

		want := `zap.Config{
  Level: zap.AtomicLevel{},
  Development: false,
  DisableCaller: false,
  DisableStacktrace: false,
  Sampling: nil,
  Encoding: "console",
  EncoderConfig: zapcore.EncoderConfig{
    MessageKey: "msg",
    LevelKey: "level",
    TimeKey: "ts",
    NameKey: "logger",
    CallerKey: "caller",
    StacktraceKey: "stacktrace",
    LineEnding: "\n",
    EncodeLevel: zapcore.CapitalColorLevelEncoder,
    EncodeTime: zapcore.ISO8601TimeEncoder,
    EncodeDuration: zapcore.SecondsDurationEncoder,
    EncodeCaller: zapcore.ShortCallerEncoder,
    EncodeName: ,
  },
  OutputPaths: []string{
    "stderr",
  },
  ErrorOutputPaths: []string{
    "stderr",
  },
  InitialFields: map[string]interface {}{
  },
}`
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("default zap config mismatch (-want +got):\n%s", diff)
		}

		if zapCfg.Level.String() != "error" {
			t.Errorf("default log level mismatch, got: %s, want: %s", zapCfg.Level.String(), "error")
		}
	})

	t.Run("env var logger configuration", func(t *testing.T) {
		defer cleanup()
		defer clearGoLogEnvVars()
		clearGoLogEnvVars()
		wantLvl := "info"
		os.Setenv(envLogging, wantLvl)
		os.Setenv(envLoggingFmt, "json")
		os.Setenv(envLoggingFile, "/tmp/golog.log")

		SetupLogging()

		got := litter.Sdump(zapCfg)

		want := `zap.Config{
  Level: zap.AtomicLevel{},
  Development: false,
  DisableCaller: false,
  DisableStacktrace: false,
  Sampling: nil,
  Encoding: "json",
  EncoderConfig: zapcore.EncoderConfig{
    MessageKey: "msg",
    LevelKey: "level",
    TimeKey: "ts",
    NameKey: "logger",
    CallerKey: "caller",
    StacktraceKey: "stacktrace",
    LineEnding: "\n",
    EncodeLevel: zapcore.CapitalColorLevelEncoder,
    EncodeTime: zapcore.ISO8601TimeEncoder,
    EncodeDuration: zapcore.SecondsDurationEncoder,
    EncodeCaller: zapcore.ShortCallerEncoder,
    EncodeName: ,
  },
  OutputPaths: []string{
    "stderr",
    "/tmp/golog.log",
  },
  ErrorOutputPaths: []string{
    "stderr",
  },
  InitialFields: map[string]interface {}{
  },
}`

		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("zap config mismatch (-want +got):\n%s", diff)
		}

		if zapCfg.Level.String() != wantLvl {
			t.Errorf("log level mismatch, got: %s, want: %s", zapCfg.Level.String(), wantLvl)
		}
	})

	t.Run("zap json config logger configuration", func(t *testing.T) {
		defer cleanup()
		defer clearGoLogEnvVars()
		clearGoLogEnvVars()
		wantLvl := "debug"
		os.Setenv(envLoggingCfg, "example_config.json")

		SetupLogging()

		got := litter.Sdump(zapCfg)

		want := `zap.Config{
  Level: zap.AtomicLevel{},
  Development: false,
  DisableCaller: false,
  DisableStacktrace: false,
  Sampling: nil,
  Encoding: "json",
  EncoderConfig: zapcore.EncoderConfig{
    MessageKey: "message",
    LevelKey: "level",
    TimeKey: "",
    NameKey: "logger",
    CallerKey: "",
    StacktraceKey: "",
    LineEnding: "",
    EncodeLevel: zapcore.LowercaseLevelEncoder,
    EncodeTime: ,
    EncodeDuration: ,
    EncodeCaller: ,
    EncodeName: ,
  },
  OutputPaths: []string{
    "stdout",
  },
  ErrorOutputPaths: nil,
  InitialFields: map[string]interface {}{
  },
}`

		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("zap config mismatch (-want +got):\n%s", diff)
		}

		if zapCfg.Level.String() != wantLvl {
			t.Errorf("log level mismatch, got: %s, want: %s", zapCfg.Level.String(), wantLvl)
		}
	})

	t.Run("zap json config with env var override", func(t *testing.T) {
		defer cleanup()
		defer clearGoLogEnvVars()
		clearGoLogEnvVars()
		wantLvl := "error"
		os.Setenv(envLoggingCfg, "example_config.json")
		// override json config with env vars
		os.Setenv(envLogging, wantLvl)
		os.Setenv(envLoggingFmt, "nocolor")
		os.Setenv(envLoggingFile, "/tmp/golog.log")
		SetupLogging()

		got := litter.Sdump(zapCfg)

		want := `zap.Config{
  Level: zap.AtomicLevel{},
  Development: false,
  DisableCaller: false,
  DisableStacktrace: false,
  Sampling: nil,
  Encoding: "console",
  EncoderConfig: zapcore.EncoderConfig{
    MessageKey: "message",
    LevelKey: "level",
    TimeKey: "",
    NameKey: "logger",
    CallerKey: "",
    StacktraceKey: "",
    LineEnding: "",
    EncodeLevel: zapcore.CapitalLevelEncoder,
    EncodeTime: ,
    EncodeDuration: ,
    EncodeCaller: ,
    EncodeName: ,
  },
  OutputPaths: []string{
    "stdout",
    "/tmp/golog.log",
  },
  ErrorOutputPaths: nil,
  InitialFields: map[string]interface {}{
  },
}`

		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("zap config mismatch (-want +got):\n%s", diff)
		}

		if zapCfg.Level.String() != wantLvl {
			t.Errorf("log level mismatch, got: %s, want: %s", zapCfg.Level.String(), wantLvl)
		}
	})
}
