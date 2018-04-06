package log

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	tracer "github.com/ipfs/go-log/tracer"
	writer "github.com/ipfs/go-log/writer"
	"github.com/stretchr/testify/assert"
)

func TestSingleEvent(t *testing.T) {
	assert := assert.New(t)

	// Set up a pipe to use as backend and log stream
	lgs, lgb := io.Pipe()
	// event logs will be written to lgb
	// event logs will be read from lgs
	writer.WriterGroup.AddWriter(lgb)

	// create a logger
	lgr := Logger("test")

	// create a root context
	ctx := context.Background()

	// start an event
	ctx = lgr.Start(ctx, "event1")

	// finish the event
	lgr.Finish(ctx)

	// decode the log event
	var ls tracer.LoggableSpan
	evtDecoder := json.NewDecoder(lgs)
	evtDecoder.Decode(&ls)

	// event name and system should be
	assert.Equal("event1", ls.Operation)
	assert.Equal("test", ls.Tags["system"])
	// greater than zero should work for now
	assert.NotZero(ls.Duration)
	assert.NotZero(ls.Start)
}
