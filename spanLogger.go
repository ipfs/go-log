// Package log is the logging library used by IPFS
// (https://github.com/ipfs/go-ipfs). It uses a modified version of
// https://godoc.org/github.com/whyrusleeping/go-logging .
package log

import (
	"bytes"
	"context"

	opentrace "github.com/opentracing/opentracing-go"
	otExt "github.com/opentracing/opentracing-go/ext"
	//otl "github.com/opentracing/opentracing-go/log"
)

// Logger retrieves an event logger by name
func NewSampleLogger(system string) SampleLogger {
	return &sampleLogger{system: system}
}

//Will the wrapper for interacting with opentracing
type SampleLogger interface {
	Start(ctx context.Context, name string) *Sample

	StartFromParentState(ctx context.Context, name string, parent []byte) *Sample
}

type sampleLogger struct {
	system string
}

//Span wrapper
type Sample struct {
	context.Context
	span opentrace.Span
}

func (sl *sampleLogger) Start(ctx context.Context, name string) *Sample {
	sampleSpan, sampleCtx := opentrace.StartSpanFromContext(ctx, name)

	out := &Sample{
		Context: sampleCtx,
		span:    sampleSpan,
	}
	out.span.SetTag("FORREST", "FORREST")
	return out

}

func (sl *sampleLogger) StartFromParentState(ctx context.Context, name string, parent []byte) *Sample {
	spanContext := deserializeContext(parent)
	span, sampleCtx := opentrace.StartSpanFromContext(ctx, name, otExt.RPCServerOption(spanContext)) //opts here

	sample := sampleCtx.(Sample)
	sample.span = span
	return &sample
}

func (s *Sample) SerializeContext() []byte {
	gTracer := opentrace.GlobalTracer()

	b := make([]byte, 0)
	carrier := bytes.NewBuffer(b)
	if err := gTracer.Inject(s.span.Context(), opentrace.Binary, carrier); err != nil {
		log.Error("Failed to inject span context to carrier")
		return nil
	}

	return carrier.Bytes()
}

func deserializeContext(bCtx []byte) opentrace.SpanContext {
	gTracer := opentrace.GlobalTracer()
	carrier := bytes.NewBuffer(bCtx)
	spanContext, err := gTracer.Extract(opentrace.Binary, carrier)
	if err != nil {
		return nil
	}
	return spanContext
}

func (s *Sample) Finish(err ...error) {
	s.span.Finish()
}
