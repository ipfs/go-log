package log

import (
	"io"
	"sync"
	"time"
)

type MirrorWriter struct {
	writers []io.WriteCloser
	lk      sync.Mutex
}

func (mw *MirrorWriter) Write(b []byte) (int, error) {
	mw.lk.Lock()
	// write to all writers, and nil out the broken ones.
	var dropped bool
	done := make(chan error, 1)
	for i, w := range mw.writers {
		go func(out chan error) {
			_, err := w.Write(b)
			out <- err
		}(done)
		select {
		case err := <-done:
			if err != nil {
				mw.writers[i].Close()
				mw.writers[i] = nil
				dropped = true
			}
		case <-time.After(time.Millisecond * 500):
			mw.writers[i].Close()
			mw.writers[i] = nil
			dropped = true

			// clear channel out
			done = make(chan error, 1)
		}
	}

	// consolidate the slice
	if dropped {
		writers := mw.writers
		mw.writers = nil
		for _, w := range writers {
			if w != nil {
				mw.writers = append(mw.writers, w)
			}
		}
	}
	mw.lk.Unlock()
	return len(b), nil
}

func (mw *MirrorWriter) AddWriter(w io.WriteCloser) {
	mw.lk.Lock()
	mw.writers = append(mw.writers, w)
	mw.lk.Unlock()
}

func (mw *MirrorWriter) Active() (active bool) {
	mw.lk.Lock()
	active = len(mw.writers) > 0
	mw.lk.Unlock()
	return
}
