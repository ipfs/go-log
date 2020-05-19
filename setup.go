package log

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	// register the global pipes sink to allow us to specify it as an output
	zap.RegisterSink("pipes", func(*url.URL) (zap.Sink, error) {
		return pipes, nil
	})

	SetupLogging(ConfigFromEnv())
}

// Logging environment variables
const (
	// IPFS_* prefixed env vars kept for backwards compatibility
	// for this release. They will not be available in the next
	// release.
	//
	// GOLOG_* env vars take precedences over IPFS_* env vars.
	envIPFSLogging    = "IPFS_LOGGING"
	envIPFSLoggingFmt = "IPFS_LOGGING_FMT"

	envLogging    = "GOLOG_LOG_LEVEL"
	envLoggingFmt = "GOLOG_LOG_FMT"

	envLoggingFile = "GOLOG_FILE" // /path/to/file
)

type Config struct {
	// Format overrides the format of the log output. Defaults to ColorizedOutput
	Format LogFormat

	// Level is the minimum enabled logging level.
	Level string

	// Stderr indicates whether logs should be written to stderr.
	Stderr bool

	// Stdout indicates whether logs should be written to stdout.
	Stdout bool

	// File is a path to a file that logs will be written to.
	File string
}

// ConfigFromEnv returns a Config with defaults populated using environment variables.
func ConfigFromEnv() Config {
	cfg := Config{
		Format: ColorizedOutput,
		Stderr: true,
	}

	format := os.Getenv(envLoggingFmt)
	if format == "" {
		format = os.Getenv(envIPFSLoggingFmt)
	}

	switch format {
	case "nocolor":
		cfg.Format = PlaintextOutput
	case "json":
		cfg.Format = JSONOutput
	}

	cfg.Level = os.Getenv(envLogging)
	if cfg.Level == "" {
		cfg.Level = os.Getenv(envIPFSLogging)
	}

	cfg.File = os.Getenv(envLoggingFile)

	return cfg
}

type LogFormat int

const (
	ColorizedOutput LogFormat = iota
	PlaintextOutput
	JSONOutput
)

// ErrNoSuchLogger is returned when the util pkg is asked for a non existant logger
var ErrNoSuchLogger = errors.New("Error: No such logger")

// loggers is the set of loggers in the system
var loggerMutex sync.RWMutex
var loggers = make(map[string]*zap.SugaredLogger)
var levels = make(map[string]zap.AtomicLevel)

var zapCfg = zap.NewProductionConfig()

// SetupLogging will initialize the logger backend and set the flags.
// TODO calling this in `init` pushes all configuration to env variables
// - move it out of `init`? then we need to change all the code (js-ipfs, go-ipfs) to call this explicitly
// - have it look for a config file? need to define what that is
func SetupLogging(cfg Config) {
	// colorful or plain
	switch cfg.Format {
	case PlaintextOutput:
		zapCfg.Encoding = "console"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	case JSONOutput:
		zapCfg.Encoding = "json"
	default:
		zapCfg.Encoding = "console"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zapCfg.Sampling = nil
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg.DisableStacktrace = true

	zapCfg.OutputPaths = []string{"pipes://"}

	if cfg.Stderr {
		zapCfg.OutputPaths = append(zapCfg.OutputPaths, "stderr")
	}
	if cfg.Stdout {
		zapCfg.OutputPaths = append(zapCfg.OutputPaths, "stdout")
	}

	// check if we log to a file
	if len(cfg.File) > 0 {
		if path, err := normalizePath(cfg.File); err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve log path '%q', logging to stderr only: %s\n", cfg.File, err)
		} else {
			zapCfg.OutputPaths = append(zapCfg.OutputPaths, path)
		}
	}

	lvl := LevelError

	if cfg.Level != "" {
		var err error
		lvl, err = LevelFromString(cfg.Level)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error setting log levels: %s\n", err)
		}
	}
	zapCfg.Level.SetLevel(zapcore.Level(lvl))

	SetAllLoggers(lvl)
}

// SetDebugLogging calls SetAllLoggers with logging.DEBUG
func SetDebugLogging() {
	SetAllLoggers(LevelDebug)
}

// SetAllLoggers changes the logging level of all loggers to lvl
func SetAllLoggers(lvl LogLevel) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	for _, l := range levels {
		l.SetLevel(zapcore.Level(lvl))
	}
}

// SetLogLevel changes the log level of a specific subsystem
// name=="*" changes all subsystems
func SetLogLevel(name, level string) error {
	lvl, err := LevelFromString(level)
	if err != nil {
		return err
	}

	// wildcard, change all
	if name == "*" {
		SetAllLoggers(lvl)
		return nil
	}

	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	// Check if we have a logger by that name
	if _, ok := levels[name]; !ok {
		return ErrNoSuchLogger
	}

	levels[name].SetLevel(zapcore.Level(lvl))

	return nil
}

// SetLogLevelRegex sets all loggers to level `l` that match expression `e`.
// An error is returned if `e` fails to compile.
func SetLogLevelRegex(e, l string) error {
	lvl, err := LevelFromString(l)
	if err != nil {
		return err
	}

	rem, err := regexp.Compile(e)
	if err != nil {
		return err
	}

	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	for name := range loggers {
		if rem.MatchString(name) {
			levels[name].SetLevel(zapcore.Level(lvl))
		}
	}
	return nil
}

// GetSubsystems returns a slice containing the
// names of the current loggers
func GetSubsystems() []string {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	subs := make([]string, 0, len(loggers))

	for k := range loggers {
		subs = append(subs, k)
	}
	return subs
}

func getLogger(name string) *zap.SugaredLogger {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	log, ok := loggers[name]
	if !ok {
		levels[name] = zap.NewAtomicLevelAt(zapCfg.Level.Level())
		cfg := zap.Config(zapCfg)
		cfg.Level = levels[name]
		newlog, err := cfg.Build()
		if err != nil {
			panic(err)
		}
		log = newlog.Named(name).Sugar()
		loggers[name] = log
	}

	return log
}
