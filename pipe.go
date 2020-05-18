package log

import (
	"io"
	"sync"

	"go.uber.org/zap/zapcore"
)

// pipes is the global instance of pipesSink
var pipes = newPipesSink()

// pipesSink is a Zap Sink that copies log output to zero or more
// connected pipe readers. Pipe readers represent in-process readers
// that are listening to the log output.
type pipesSink struct {
	mu       sync.RWMutex
	combined zapcore.WriteSyncer
	writers  map[*PipeReader]zapcore.WriteSyncer
}

func newPipesSink() *pipesSink {
	s := &pipesSink{
		writers: make(map[*PipeReader]zapcore.WriteSyncer),
	}
	// Initially this will result in a WriteSyncer that wraps io.Discard
	s.buildCombinedWriter()
	return s
}

func (s *pipesSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We can ignore errors since these in-memory pipes won't error on
	// close and, besides, we are most likely closing the process.
	for pr := range s.writers {
		_ = pr.r.Close()
	}

	return nil
}

func (s *pipesSink) Sync() error {
	s.mu.RLock()
	err := s.combined.Sync()
	s.mu.RUnlock()
	return err
}

func (s *pipesSink) Write(b []byte) (int, error) {
	s.mu.RLock()
	n, err := s.combined.Write(b)
	s.mu.RUnlock()
	return n, err
}

// NewReader registers a new pipe reader and rebuilds the
// combined Zap WriteSyncer to include it.
func (s *pipesSink) NewReader() *PipeReader {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, w := io.Pipe()
	pr := &PipeReader{
		r:    r,
		sink: s,
	}

	ws := zapcore.AddSync(w)
	s.writers[pr] = ws

	s.buildCombinedWriter()

	return pr
}

// note the caller must hold s.mu before calling buildCombinedWriter
func (s *pipesSink) buildCombinedWriter() {
	current := make([]zapcore.WriteSyncer, 0, len(s.writers))
	for _, ws := range s.writers {
		current = append(current, ws)
	}
	s.combined = zapcore.NewMultiWriteSyncer(current...)
}

// RemoveReader unregisters a pipe reader and rebuilds the
// combined Zap WriteSyncer to exclude it.
func (s *pipesSink) RemoveReader(p *PipeReader) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.writers, p)
	s.buildCombinedWriter()
}

// A PipeReader is a reader that reads from the logger. It is synchronous
// so blocking on read will affect logging performance.
type PipeReader struct {
	sink *pipesSink
	r    *io.PipeReader
}

// Read implements the standard Read interface
func (p *PipeReader) Read(data []byte) (int, error) {
	return p.r.Read(data)
}

// Close unregisters the reader from the logger.
func (p *PipeReader) Close() error {
	p.sink.RemoveReader(p)
	return p.r.Close()
}

// NewPipeReader creates a new in-memory reader that reads from the logger.
// The caller must call Close on the returned reader when done.
func NewPipeReader() *PipeReader {
	return pipes.NewReader()
}
