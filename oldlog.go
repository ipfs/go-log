package log

import (
	"fmt"
	tracer "github.com/ipfs/go-log/tracer"
	lwriter "github.com/ipfs/go-log/writer"
	"os"

	opentrace "github.com/opentracing/opentracing-go"

	log2 "github.com/ipfs/go-log/v2"

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

	envLogging    = "GOLOG_LOG_LEVEL"
	envLoggingFmt = "GOLOG_LOG_FMT"

	envLoggingFile = "GOLOG_FILE"         // /path/to/file
	envTracingFile = "GOLOG_TRACING_FILE" // /path/to/file
)

// SetupLogging will initialize the logger backend and set the flags.
// TODO calling this in `init` pushes all configuration to env variables
// - move it out of `init`? then we need to change all the code (js-ipfs, go-ipfs) to call this explicitly
// - have it look for a config file? need to define what that is
var zapCfg = zap.NewProductionConfig()

func SetupLogging() {
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
	lvl := LevelError

	logenv := os.Getenv(envLogging)
	if logenv == "" {
		logenv = os.Getenv(envIPFSLogging)
	}

	if logenv != "" {
		var err error
		lvl, err = LevelFromString(logenv)
		if err != nil {
			fmt.Println("error setting log levels", err)
		}
	}
	zapCfg.Level.SetLevel(zapcore.Level(lvl))

	// TracerPlugins are instantiated after this, so use loggable tracer
	// by default, if a TracerPlugin is added it will override this
	lgblRecorder := tracer.NewLoggableRecorder()
	lgblTracer := tracer.New(lgblRecorder)
	opentrace.SetGlobalTracer(lgblTracer)

	SetAllLoggers(lvl)

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
	SetAllLoggers(LevelDebug)
}

// SetAllLoggers changes the logging level of all loggers to lvl
func SetAllLoggers(lvl LogLevel) {
	lvl2 := log2.LogLevel(lvl)
	log2.SetAllLoggers(lvl2)
}

// SetLogLevel changes the log level of a specific subsystem
// name=="*" changes all subsystems
func SetLogLevel(name, level string) error {
	return log2.SetLogLevel(name, level)
}

// SetLogLevelRegex sets all loggers to level `l` that match expression `e`.
// An error is returned if `e` fails to compile.
func SetLogLevelRegex(e, l string) error {
	return log2.SetLogLevelRegex(e, l)
}

// GetSubsystems returns a slice containing the
// names of the current loggers
func GetSubsystems() []string {
	return log2.GetSubsystems()
}
