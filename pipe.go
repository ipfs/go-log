package log

import (
	"io"

	"go.uber.org/zap/zapcore"
)

// A PipeReader is a reader that reads from the logger. It is synchronous
// so blocking on read will affect logging performance.
type PipeReader struct {
	r    *io.PipeReader
	core zapcore.Core
}

// Read implements the standard Read interface
func (p *PipeReader) Read(data []byte) (int, error) {
	return p.r.Read(data)
}

// Close unregisters the reader from the logger.
func (p *PipeReader) Close() error {
	if p.core != nil {
		loggerCore.DeleteCore(p.core)
	}
	return p.r.Close()
}

// NewPipeReader creates a new in-memory reader that reads from all loggers
// The caller must call Close on the returned reader when done.
func NewPipeReader(opts ...PipeReaderOption) *PipeReader {
	loggerMutex.Lock()
	opt := pipeReaderOptions{
		format: primaryFormat,
		level:  primaryLevel,
	}
	loggerMutex.Unlock()

	for _, o := range opts {
		o.setOption(&opt)
	}

	r, w := io.Pipe()

	p := &PipeReader{
		r:    r,
		core: newCore(opt.format, zapcore.AddSync(w), opt.level),
	}

	loggerCore.AddCore(p.core)

	return p
}

type pipeReaderOptions struct {
	format LogFormat
	level  LogLevel
}

type PipeReaderOption interface {
	setOption(*pipeReaderOptions)
}

type pipeReaderOptionFunc func(*pipeReaderOptions)

func (p pipeReaderOptionFunc) setOption(o *pipeReaderOptions) {
	p(o)
}

// PipeFormat sets the output format of the pipe reader
func PipeFormat(format LogFormat) PipeReaderOption {
	return pipeReaderOptionFunc(func(o *pipeReaderOptions) {
		o.format = format
	})
}

// PipeLevel sets the log level of logs sent to the pipe reader.
func PipeLevel(level LogLevel) PipeReaderOption {
	return pipeReaderOptionFunc(func(o *pipeReaderOptions) {
		o.level = level
	})
}
