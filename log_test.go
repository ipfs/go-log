package log

import (
	"context"
	"encoding/json"
	"errors"
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

func TestSingleEventWithErr(t *testing.T) {
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
	lgr.FinishWithErr(ctx, errors.New("rawer im an error"))

	// decode the log event
	var ls tracer.LoggableSpan
	evtDecoder := json.NewDecoder(lgs)
	evtDecoder.Decode(&ls)

	// event name and system should be
	assert.Equal("event1", ls.Operation)
	assert.Equal("test", ls.Tags["system"])
	assert.Equal(true, ls.Tags["error"])
	assert.Contains(ls.Logs[0].Field[0].Value, "rawer im an error")
	// greater than zero should work for now
	assert.NotZero(ls.Duration)
	assert.NotZero(ls.Start)
}

func TestEventWithTag(t *testing.T) {
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
	lgr.SetTag(ctx, "tk", "tv")

	// finish the event
	lgr.Finish(ctx)

	// decode the log event
	var ls tracer.LoggableSpan
	evtDecoder := json.NewDecoder(lgs)
	evtDecoder.Decode(&ls)

	// event name and system should be
	assert.Equal("event1", ls.Operation)
	assert.Equal("test", ls.Tags["system"])
	assert.Equal("tv", ls.Tags["tk"])
	// greater than zero should work for now
	assert.NotZero(ls.Duration)
	assert.NotZero(ls.Start)
}

func TestMultiEvent(t *testing.T) {
	assert := assert.New(t)

	// Set up a pipe to use as backend and log stream
	lgs, lgb := io.Pipe()
	// event logs will be written to lgb
	// event logs will be read from lgs
	writer.WriterGroup.AddWriter(lgb)
	evtDecoder := json.NewDecoder(lgs)

	// create a logger
	lgr := Logger("test")

	// create a root context
	ctx := context.Background()

	ctx = lgr.Start(ctx, "root")

	doEvent(ctx, "e1", lgr)
	doEvent(ctx, "e2", lgr)

	lgr.Finish(ctx)

	e1 := getEvent(evtDecoder)
	assert.Equal("e1", e1.Operation)
	assert.Equal("test", e1.Tags["system"])
	assert.NotZero(e1.Duration)
	assert.NotZero(e1.Start)

	// I hope your clocks work...
	e2 := getEvent(evtDecoder)
	assert.Equal("e2", e2.Operation)
	assert.Equal("test", e2.Tags["system"])
	assert.NotZero(e2.Duration)
	assert.True(e1.Start.Nanosecond() < e2.Start.Nanosecond())

	er := getEvent(evtDecoder)
	assert.Equal("root", er.Operation)
	assert.Equal("test", er.Tags["system"])
	assert.True(er.Duration.Nanoseconds() > e1.Duration.Nanoseconds()+e2.Duration.Nanoseconds())
	assert.NotZero(er.Start)

}

func TestEventSerialization(t *testing.T) {
	assert := assert.New(t)

	// Set up a pipe to use as backend and log stream
	lgs, lgb := io.Pipe()
	// event logs will be written to lgb
	// event logs will be read from lgs
	writer.WriterGroup.AddWriter(lgb)
	evtDecoder := json.NewDecoder(lgs)

	// create a logger
	lgr := Logger("test")

	// start an event
	sndctx := lgr.Start(context.Background(), "send")

	// **imagine** that we are putting `bc` (byte context) into a protobuf message
	// and send the message to another peer on the network
	bc, err := lgr.SerializeContext(sndctx)
	assert.NoError(err)

	// now  **imagine** some peer getting a protobuf message and extracting
	// `bc` from the message to continue the operation
	rcvctx, err := lgr.StartFromParentState(context.Background(), "recv", bc)
	assert.NoError(err)

	// at some point the sender completes their operation
	lgr.Finish(sndctx)
	e := getEvent(evtDecoder)
	assert.Equal("send", e.Operation)
	assert.Equal("test", e.Tags["system"])
	assert.NotZero(e.Start)
	assert.NotZero(e.Start)

	// and then the receiver finishes theirs
	lgr.Finish(rcvctx)
	e = getEvent(evtDecoder)
	assert.Equal("recv", e.Operation)
	assert.Equal("test", e.Tags["system"])
	assert.NotZero(e.Start)
	assert.NotZero(e.Start)

}

func doEvent(ctx context.Context, name string, el EventLogger) context.Context {
	ctx = el.Start(ctx, name)
	defer func() {
		el.Finish(ctx)
	}()
	return ctx
}

func getEvent(ed *json.Decoder) tracer.LoggableSpan {
	// decode the log event
	var ls tracer.LoggableSpan
	ed.Decode(&ls)
	return ls
}
