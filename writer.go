package log

import (
	"fmt"
	"io"
	"sync"
)

var MaxWriterBuffer = 512 * 1024

var log = Logger("eventlog")

type MirrorWriter struct {
	active   bool
	activelk sync.Mutex

	// channel for incoming writers
	writerAdd chan io.WriteCloser

	// slices of writer/sync-channel pairs
	writers []*bufWriter

	// synchronization channel for incoming writes
	msgSync chan []byte
}

type writerSync struct {
	w  io.WriteCloser
	br chan []byte
}

func NewMirrorWriter() *MirrorWriter {
	mw := &MirrorWriter{
		msgSync:   make(chan []byte, 64), // sufficiently large buffer to avoid callers waiting
		writerAdd: make(chan io.WriteCloser),
	}

	go mw.logRoutine()

	return mw
}

func (mw *MirrorWriter) Write(b []byte) (int, error) {
	mw.msgSync <- b
	return len(b), nil
}

func (mw *MirrorWriter) Close() error {
	// it is up to the caller to ensure that write is not called during or
	// after close is called.
	close(mw.msgSync)
	return nil
}

func (mw *MirrorWriter) logRoutine() {
	// rebind to avoid races on nilling out struct fields
	msgSync := mw.msgSync
	writerAdd := mw.writerAdd

	for {
		select {
		case b, ok := <-msgSync:
			if !ok {
				return
			}
			// write to all writers
			dropped := mw.broadcastMessage(b)

			// consolidate the slice
			if dropped {
				mw.clearDeadWriters()
			}
		case w := <-writerAdd:
			mw.writers = append(mw.writers, newBufWriter(w))

			mw.activelk.Lock()
			mw.active = true
			mw.activelk.Unlock()
		}
	}
}

// broadcastMessage sends the given message to every writer
// if any writer is killed during the send, 'true' is returned
func (mw *MirrorWriter) broadcastMessage(b []byte) bool {
	var dropped bool
	for i, w := range mw.writers {
		_, err := w.Write(b)
		if err != nil {
			mw.writers[i] = nil
			dropped = true
		}
	}
	return dropped
}

func (mw *MirrorWriter) clearDeadWriters() {
	writers := mw.writers
	mw.writers = nil
	for _, w := range writers {
		if w != nil {
			mw.writers = append(mw.writers, w)
		}
	}
	if len(mw.writers) == 0 {
		mw.activelk.Lock()
		mw.active = false
		mw.activelk.Unlock()
	}
}

func (mw *MirrorWriter) AddWriter(w io.WriteCloser) {
	mw.writerAdd <- w
}

func (mw *MirrorWriter) Active() (active bool) {
	mw.activelk.Lock()
	active = mw.active
	mw.activelk.Unlock()
	return
}

func newBufWriter(w io.WriteCloser) *bufWriter {
	bw := &bufWriter{
		writer:   w,
		incoming: make(chan []byte, 1),
	}

	go bw.loop()
	return bw
}

type bufWriter struct {
	writer io.WriteCloser

	incoming chan []byte

	deathLock sync.Mutex
	dead      bool
}

var errDeadWriter = fmt.Errorf("writer is dead")

func (bw *bufWriter) Write(b []byte) (int, error) {
	bw.deathLock.Lock()
	dead := bw.dead
	bw.deathLock.Unlock()
	if dead {
		if bw.incoming != nil {
			close(bw.incoming)
			bw.incoming = nil
		}
		return 0, errDeadWriter
	}

	bw.incoming <- b
	return len(b), nil
}

func (bw *bufWriter) die() {
	bw.deathLock.Lock()
	bw.dead = true
	bw.writer.Close()
	bw.deathLock.Unlock()
}

func (bw *bufWriter) loop() {
	bufsize := 0
	bufBase := make([][]byte, 0, 16) // some initial memory
	buffered := bufBase
	nextCh := make(chan []byte)

	var nextMsg []byte
	var send chan []byte

	go func() {
		for b := range nextCh {
			_, err := bw.writer.Write(b)
			if err != nil {
				log.Info("eventlog write error: %s", err)
				bw.die()
				return
			}
		}
	}()

	// collect and buffer messages
	incoming := bw.incoming
	for {
		select {
		case b, ok := <-incoming:
			if !ok {
				return
			}
			if len(buffered) == 0 {
				nextMsg = b
				send = nextCh
			} else {
				bufsize += len(b)
				buffered = append(buffered, b)
				if bufsize > MaxWriterBuffer {
					// if we have too many messages buffered, kill the writer
					bw.die()
					close(nextCh)
					// explicity keep going here to drain incoming
				}
			}
		case send <- nextMsg:
			// if 'send' is equal to nil, this case will never trigger.
			// Taking advantage of that, when we have sent all of our buffered
			// messages, we nil out the channel until more arrive. This way we
			// can 'turn off' the writer until there is more to be written.
			if len(buffered) > 0 {
				nextMsg = buffered[0]
				buffered = buffered[1:]
				bufsize -= len(nextMsg)
			}

			if len(buffered) == 0 {
				// reset slice position
				buffered = bufBase[:0]

				nextMsg = nil
				send = nil
			}
		}
	}
}
