// Package log is the logging library used by IPFS
// (https://github.com/ipfs/go-ipfs). It uses a modified version of
// https://godoc.org/github.com/whyrusleeping/go-logging .
package log

import (
	"bytes"
	"context"
	"path"
	"runtime"
	"time"

	opentrace "github.com/opentracing/opentracing-go"
	otExt "github.com/opentracing/opentracing-go/ext"
)

var log = Logger("eventlog")

// StandardLogger provides API compatibility with standard printf loggers
// eg. go-logging
type StandardLogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

// EventLogger extends the StandardLogger interface to allow for log items
// containing structured metadata
type EventLogger interface {
	StandardLogger

	// Start starts an opentracing span with `name`, using
	// any Span found within `ctx` as a ChildOfRef. If no such parent could be
	// found, Start creates a root (parentless) Span.
	//
	// The return value is a context.Context object built around the
	// returned Span.
	//
	// Example usage:
	//
	//    SomeFunction(ctx context.Context, ...) {
	//        ctx := log.Start(ctx, "SomeFunction")
	//        defer log.Finish(ctx)
	//        ...
	//    }
	Start(ctx context.Context, name string) context.Context

	// StartFromParentState starts an opentracing span with `name`, using
	// any Span found within `ctx` as a ChildOfRef. If no such parent could be
	// found, StartSpanFromParentState creates a root (parentless) Span.
	//
	// StartFromParentState will attempt to deserialize a SpanContext from `parent`,
	// using any Span found within to continue the trace
	//
	// The return value is a context.Context object built around the
	// returned Span.
	//
	// An error is returned when `parent` cannot be deserialized to a SpanContext
	//
	// Example usage:
	//
	//    SomeFunction(ctx context.Context, bParent []byte) {
	//        ctx := log.StartFromParentState(ctx, "SomeFunction", bParent)
	//        defer log.Finish(ctx)
	//        ...
	//    }
	StartFromParentState(ctx context.Context, name string, parent []byte) (context.Context, error)

	// Finish completes the span associated with `ctx`.
	//
	// Finish() must be the last call made to any span instance, and to do
	// otherwise leads to undefined behavior.
	// Finish will do its best to notify (log) when used in correctly
	//		.e.g called twice, or called on a spanless `ctx`
	Finish(ctx context.Context)

	// FinishWithErr completes the span associated with `ctx` and also calls
	// SetErr if `err` is non-nil
	//
	// FinishWithErr() must be the last call made to any span instance, and to do
	// otherwise leads to undefined behavior.
	// FinishWithErr will do its best to notify (log) when used in correctly
	//		.e.g called twice, or called on a spanless `ctx`
	FinishWithErr(ctx context.Context, err error)

	// SetErr tags the span associated with `ctx` to reflect an error occured, and
	// logs the value `err` under key `error`.
	SetErr(ctx context.Context, err error)

	// LogKV records key:value logging data about an event stored in `ctx`
	// Eexample:
	//    log.LogKV(
	//        "error", "resolve failure",
	//        "type", "cache timeout",
	//        "waited.millis", 1500)
	LogKV(ctx context.Context, alternatingKeyValues ...interface{})

	// SetTag tags key `k` and value `v` on the span associated with `ctx`
	SetTag(ctx context.Context, key string, value interface{})

	// SetTags tags keys from the `tags` maps on the span associated with `ctx`
	// Example:
	//    log.SetTags(ctx, map[string]{
	//		"type": bizStruct,
	//      "request": req,
	//		})
	SetTags(ctx context.Context, tags map[string]interface{})

	// SerializeContext takes the SpanContext instance stored in `ctx` and Seralizes
	// it to bytes. An error is returned if the `ctx` cannot be serialized to
	// a bytes array
	SerializeContext(ctx context.Context) ([]byte, error)
}

// Logger retrieves an event logger by name
func Logger(system string) EventLogger {

	// TODO if we would like to adjust log levels at run-time. Store this event
	// logger in a map (just like the util.Logger impl)
	if len(system) == 0 {
		setuplog := getLogger("setup-logger")
		setuplog.Warning("Missing name parameter")
		system = "undefined"
	}

	logger := getLogger(system)

	return &eventLogger{system: system, StandardLogger: logger}
}

// eventLogger implements the EventLogger and wraps a go-logging Logger
type eventLogger struct {
	StandardLogger

	system string
	// TODO add log-level
}

func (el *eventLogger) Start(ctx context.Context, operationName string) context.Context {
	span, ctx := opentrace.StartSpanFromContext(ctx, operationName)
	span.SetTag("system", el.system)
	return ctx
}

func (el *eventLogger) StartFromParentState(ctx context.Context, operationName string, parent []byte) (context.Context, error) {
	sc, err := deserializeContext(parent)
	if err != nil {
		return nil, err
	}

	//TODO RPCServerOption is probably not the best tag, as this is likely from a peer
	span, ctx := opentrace.StartSpanFromContext(ctx, operationName, otExt.RPCServerOption(sc))
	span.SetTag("system", el.system)
	return ctx, nil
}

func (el *eventLogger) SerializeContext(ctx context.Context) ([]byte, error) {
	gTracer := opentrace.GlobalTracer()
	b := make([]byte, 0)
	carrier := bytes.NewBuffer(b)
	span := opentrace.SpanFromContext(ctx)
	if err := gTracer.Inject(span.Context(), opentrace.Binary, carrier); err != nil {
		return nil, err
	}
	return carrier.Bytes(), nil
}

func (el *eventLogger) LogKV(ctx context.Context, alternatingKeyValues ...interface{}) {
	span := opentrace.SpanFromContext(ctx)
	if span == nil {
		_, file, line, _ := runtime.Caller(1)
		log.Errorf("LogKV with no Span in context called on %s:%d", path.Base(file), line)
		return
	}
	span.LogKV(alternatingKeyValues...)
}

func (el *eventLogger) SetTag(ctx context.Context, k string, v interface{}) {
	span := opentrace.SpanFromContext(ctx)
	if span == nil {
		_, file, line, _ := runtime.Caller(1)
		log.Errorf("SetTag with no Span in context called on %s:%d", path.Base(file), line)
		return
	}
	span.SetTag(k, v)
}

func (el *eventLogger) SetTags(ctx context.Context, tags map[string]interface{}) {
	span := opentrace.SpanFromContext(ctx)
	if span == nil {
		_, file, line, _ := runtime.Caller(1)
		log.Errorf("SetTags with no Span in context called on %s:%d", path.Base(file), line)
		return
	}
	for k, v := range tags {
		span.SetTag(k, v)
	}
}

func (el *eventLogger) setErr(ctx context.Context, err error, skip int) {
	span := opentrace.SpanFromContext(ctx)
	if span == nil {
		_, file, line, _ := runtime.Caller(skip)
		log.Errorf("SetErr with no Span in context called on %s:%d", path.Base(file), line)
		return
	}
	if err == nil {
		return
	}

	otExt.Error.Set(span, true)
	span.LogKV("error", err.Error())
}

func (el *eventLogger) SetErr(ctx context.Context, err error) {
	el.setErr(ctx, err, 1)
}

func (el *eventLogger) Finish(ctx context.Context) {
	span := opentrace.SpanFromContext(ctx)
	if span == nil {
		_, file, line, _ := runtime.Caller(1)
		log.Errorf("Finish with no Span in context called on %s:%d", path.Base(file), line)
		return
	}
	span.Finish()
}

func (el *eventLogger) FinishWithErr(ctx context.Context, err error) {
	el.setErr(ctx, err, 2)
	el.Finish(ctx)
}

func deserializeContext(bCtx []byte) (opentrace.SpanContext, error) {
	gTracer := opentrace.GlobalTracer()
	carrier := bytes.NewReader(bCtx)
	spanContext, err := gTracer.Extract(opentrace.Binary, carrier)
	if err != nil {
		log.Warning("Failed to deserialize context %s", err)
		return nil, err
	}
	return spanContext, nil
}

type activeEventKeyType struct{}

var activeEventKey = activeEventKeyType{}

// FormatRFC3339 returns the given time in UTC with RFC3999Nano format.
func FormatRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
