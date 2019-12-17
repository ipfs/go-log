package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	SetupLogging()
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

	envLogging     = "GOLOG_LOG_LEVEL"
	envLoggingFmt  = "GOLOG_LOG_FMT"
	envLoggingCfg  = "GOLOG_LOG_CONFIG" // /path/to/file
	envLoggingFile = "GOLOG_FILE"       // /path/to/file

	// envTracingFile is deprecated.
	envTracingFile = "GOLOG_TRACING_FILE" //nolint: unused
)

// ErrNoSuchLogger is returned when the util pkg is asked for a non existant logger
var ErrNoSuchLogger = errors.New("Error: No such logger")

// loggers is the set of loggers in the system
var loggerMutex sync.RWMutex
var loggers = make(map[string]*zap.SugaredLogger)
var levels = make(map[string]zap.AtomicLevel)

// fields contains key/value pairs that will be added to all
// loggers.
var fields = make([]interface{}, 0)

var zapCfg = zap.NewProductionConfig()

// SetupLogging will initialize the logger backend and set the flags.
func SetupLogging() {
	var jsonCfg []byte
	var err error
	jsonCfgFile, ok := os.LookupEnv(envLoggingCfg)
	if ok {
		jsonCfg, err = ioutil.ReadFile(jsonCfgFile)
		if err != nil {
			fmt.Println(fmt.Errorf("failed to read json config file: %w", err))
			fmt.Printf("initializing go-log with default configuration")
		}
		setupLogging(jsonCfg)
	} else {
		setupLogging(nil)
	}
}

func setupLogging(jsonCfg []byte) {
	if jsonCfg != nil {
		var jsonZapCfg zap.Config
		if err := json.Unmarshal(jsonCfg, &jsonZapCfg); err != nil {
			fmt.Printf("failed to unmarshal zap json config")
			panic(err)
		}
		zapCfg = jsonZapCfg
	} else {
		// the following config options are not exposed via env vars
		// so they are used when no json config is provided.
		zapCfg.Sampling = nil
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapCfg.OutputPaths = []string{"stderr"}
		zapCfg.Encoding = "console"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapCfg.Level.SetLevel(zapcore.Level(LevelError))
	}

	// if the following env vars are defined, they will override any
	// values that are set in the json config.
	loggingFmt := os.Getenv(envLoggingFmt)
	if loggingFmt == "" {
		loggingFmt = os.Getenv(envIPFSLoggingFmt)
	}
	// colorful or plain
	switch loggingFmt {
	case "nocolor":
		zapCfg.Encoding = "console"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	case "json":
		zapCfg.Encoding = "json"
	}

	// check if we log to a file
	if logfp := os.Getenv(envLoggingFile); len(logfp) > 0 {
		zapCfg.OutputPaths = append(zapCfg.OutputPaths, logfp)
	}

	logenv := os.Getenv(envLogging)
	if logenv == "" {
		logenv = os.Getenv(envIPFSLogging)
	}

	// set the backend(s)
	var lvl LogLevel
	if logenv != "" {
		var err error
		lvl, err = LevelFromString(logenv)
		if err != nil {
			fmt.Println("error setting log levels", err)
		}
		zapCfg.Level.SetLevel(zapcore.Level(lvl))
		SetAllLoggers(lvl)
	}
}

// SetFieldOnAllLoggers adds the provided key value args as fields to
// all loggers.
// Must be called before Logger, i.e. in an init() function.
// SetFieldOnAllLoggers will panic if the length of args is not even.
func SetFieldsOnAllLoggers(args ...interface{}) {
	if len(args)%2 != 0 {
		panic(fmt.Errorf("SetFieldOnAllLoggers: length of args must be an even number as it represents key/value pairs: len %d", len(args)))
	}

	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	fields = append(fields, args...)
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
		zapCfg.Level = levels[name]
		newlog, err := zapCfg.Build()
		if err != nil {
			panic(err)
		}
		log = newlog.Named(name).Sugar().With(fields...)
		loggers[name] = log
	}

	return log
}

// Cleanup is for testing purposes only.
// Cleanup resets the package state.
func Cleanup() {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	loggers = make(map[string]*zap.SugaredLogger)
	levels = make(map[string]zap.AtomicLevel)
	fields = make([]interface{}, 0)
}
