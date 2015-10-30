package log

import (
	"io"
	"sync"
)

var MaxWriterBuffer = 16 * 1024 * 1024

var log = Logger("eventlog")

type MirrorWriter struct {
	active   bool
	activelk sync.Mutex

	// channel for incoming writers
	writerAdd chan io.WriteCloser

	// slices of writer/sync-channel pairs
	writers []*writerSync

	// synchronization channel for incoming writes
	msgSync chan []byte

	// channel for dead writers to notify the MirrorWriter
	deadWriter chan io.WriteCloser
}

type writerSync struct {
	w  io.WriteCloser
	br chan []byte
}

func NewMirrorWriter() *MirrorWriter {
	mw := &MirrorWriter{
		msgSync:    make(chan []byte, 64), // sufficiently large buffer to avoid callers waiting
		writerAdd:  make(chan io.WriteCloser),
		deadWriter: make(chan io.WriteCloser),
	}

	go mw.logRoutine()

	return mw
}

func (mw *MirrorWriter) Write(b []byte) (int, error) {
	mw.msgSync <- b
	return len(b), nil
}

func (mw *MirrorWriter) handleWriter(w io.WriteCloser, msgs <-chan []byte) {
	bufsize := 0
	bufBase := make([][]byte, 0, 16) // some initial memory
	buffered := bufBase
	nextCh := make(chan []byte)

	var nextMsg []byte
	var send chan []byte

	go func() {
		for b := range nextCh {
			_, err := w.Write(b)
			if err != nil {
				log.Info("eventlog write error: %s", err)
				return
			}
		}
	}()

	// collect and buffer messages
	for {
		select {
		case b := <-msgs:
			if len(buffered) == 0 {
				nextMsg = b
				send = nextCh
			} else {
				bufsize += len(b)
				buffered = append(buffered, b)
				if bufsize > MaxWriterBuffer {
					// if we have too many messages buffered, kill the writer
					w.Close()
					mw.deadWriter <- w
					close(nextCh)
					return
				}
			}
		case send <- nextMsg:
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

func (mw *MirrorWriter) logRoutine() {
	for {
		select {
		case b := <-mw.msgSync:
			// write to all writers
			dropped := mw.broadcastMessage(b)

			// consolidate the slice
			if dropped {
				mw.clearDeadWriters()
			}
		case w := <-mw.writerAdd:
			brchan := make(chan []byte, 1) // buffered for absent-handoffs to not cause delays
			mw.writers = append(mw.writers, &writerSync{
				w:  w,
				br: brchan,
			})
			go mw.handleWriter(w, brchan)

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
		if w == nil {
			// if the next writer was killed before we got
			// to it move on
			continue
		}

		for sending := true; sending; {
			// loop until we send the message, or the current writer is killed
			select {
			case w.br <- b:
				// success!
				sending = false

			case dw := <-mw.deadWriter:
				// some writer was killed while waiting here
				for j, w := range mw.writers {
					if w.w == dw {
						mw.writers[j] = nil
						if i == j {
							sending = false
						}
					}
				}
				dropped = true
			}
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
