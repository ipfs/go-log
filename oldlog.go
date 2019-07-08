package log

import (
	"errors"
	"fmt"
	"os"
	"sync"

	tracer "github.com/ipfs/go-log/tracer"
	lwriter "github.com/ipfs/go-log/writer"

	opentrace "github.com/opentracing/opentracing-go"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	SetupLogging()
}

var ansiGray = "\033[0;37m"
var ansiBlue = "\033[0;34m"

// LogFormats defines formats for logging (i.e. "color")
var LogFormats = map[string]string{
	"nocolor": "%{time:2006-01-02 15:04:05.000000} %{level} %{module} %{shortfile}: %{message}",
	"color": ansiGray + "%{time:15:04:05.000} %{color}%{level:5.5s} " + ansiBlue +
		"%{module:10.10s}: %{color:reset}%{message} " + ansiGray + "%{shortfile}%{color:reset}",
}

var defaultLogFormat = "color"

// Logging environment variables
const (
	// TODO these env names should be more general, IPFS is not the only project to
	// use go-log
	envLogging    = "IPFS_LOGGING"
	envLoggingFmt = "IPFS_LOGGING_FMT"

	envLoggingFile = "GOLOG_FILE"         // /path/to/file
	envTracingFile = "GOLOG_TRACING_FILE" // /path/to/file
)

// ErrNoSuchLogger is returned when the util pkg is asked for a non existant logger
var ErrNoSuchLogger = errors.New("Error: No such logger")

// loggers is the set of loggers in the system
var loggerMutex sync.RWMutex
var loggers = make(map[string]*zap.SugaredLogger)
var levels = make(map[string]zap.AtomicLevel)

// SetupLogging will initialize the logger backend and set the flags.
// TODO calling this in `init` pushes all configuration to env variables
// - move it out of `init`? then we need to change all the code (js-ipfs, go-ipfs) to call this explicitly
// - have it look for a config file? need to define what that is
var zapCfg = zap.NewProductionConfig()

func SetupLogging() {

	// colorful or plain
	switch os.Getenv(envLoggingFmt) {
	case "nocolor":
		zapCfg.Encoding = "console"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	case "json":
		zapCfg.Encoding = "json"
	default:
		zapCfg.Encoding = "console"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zapCfg.Sampling = nil
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	zapCfg.OutputPaths = []string{"stderr"}
	// check if we log to a file
	if logfp := os.Getenv(envLoggingFile); len(logfp) > 0 {
		zapCfg.OutputPaths = append(zapCfg.OutputPaths, logfp)
	}

	// set the backend(s)
	lvl := new(zapcore.Level)
	*lvl = zapcore.ErrorLevel

	if logenv := os.Getenv(envLogging); logenv != "" {
		err := lvl.Set(logenv)
		if err != nil {
			fmt.Println("error setting log levels", err)
		}
	}
	zapCfg.Level.SetLevel(*lvl)

	// TracerPlugins are instantiated after this, so use loggable tracer
	// by default, if a TracerPlugin is added it will override this
	lgblRecorder := tracer.NewLoggableRecorder()
	lgblTracer := tracer.New(lgblRecorder)
	opentrace.SetGlobalTracer(lgblTracer)

	SetAllLoggers(*lvl)

	if tracingfp := os.Getenv(envTracingFile); len(tracingfp) > 0 {
		f, err := os.Create(tracingfp)
		if err != nil {
			log.Error("failed to create tracing file: %s", tracingfp)
		} else {
			lwriter.WriterGroup.AddWriter(f)
		}
	}
}

// SetDebugLogging calls SetAllLoggers with logging.DEBUG
func SetDebugLogging() {
	SetAllLoggers(zapcore.DebugLevel)
}

// SetAllLoggers changes the logging.Level of all loggers to lvl
func SetAllLoggers(lvl zapcore.Level) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	for _, l := range levels {
		l.SetLevel(lvl)
	}
}

// SetLogLevel changes the log level of a specific subsystem
// name=="*" changes all subsystems
func SetLogLevel(name, level string) error {
	lvl := new(zapcore.Level)
	err := lvl.Set(level)
	if err != nil {
		return err
	}

	// wildcard, change all
	if name == "*" {
		SetAllLoggers(*lvl)
		return nil
	}

	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	// Check if we have a logger by that name
	if _, ok := levels[name]; !ok {
		return ErrNoSuchLogger
	}

	levels[name].SetLevel(*lvl)

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
