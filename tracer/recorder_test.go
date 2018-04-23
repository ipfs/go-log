package loggabletracer

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	writer "github.com/ipfs/go-log/writer"
	opentrace "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
)

func TestSpanRecorder(t *testing.T) {
	assert := assert.New(t)

	// Set up a writer to send spans to
	pr, pw := io.Pipe()
	writer.WriterGroup.AddWriter(pw)

	// create a span recorder
	recorder := NewLoggableRecorder()

	// generate a span
	var apiRecorder SpanRecorder = recorder
	rt := opentrace.Tags{
		"key": "value",
	}
	rs := RawSpan{
		Context:   SpanContext{},
		Operation: "test-span",
		Start:     time.Now(),
		Duration:  -1,
		Tags:      rt,
	}

	// record the span
	apiRecorder.RecordSpan(rs)

	// decode the LoggableSpan from
	var ls LoggableSpan
	evtDecoder := json.NewDecoder(pr)
	evtDecoder.Decode(&ls)

	// validate
	assert.Equal(rs.Operation, ls.Operation)
	assert.Equal(rs.Duration, ls.Duration)
	assert.Equal(rs.Start.Nanosecond(), ls.Start.Nanosecond())
	assert.Equal(rs.Tags, ls.Tags)

}
