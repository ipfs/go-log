package log

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestNewPipeReader(t *testing.T) {
	log := getLogger("test")

	var wg sync.WaitGroup
	wg.Add(1)

	r := NewPipeReader()

	buf := &bytes.Buffer{}
	go func() {
		defer wg.Done()
		if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	log.Error("scooby")
	r.Close()
	wg.Wait()

	if !strings.Contains(buf.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

}

func TestNewPipeReaderFormat(t *testing.T) {
	log := getLogger("test")

	var wg sync.WaitGroup
	wg.Add(1)

	r := NewPipeReader(PipeFormat(PlaintextOutput))

	buf := &bytes.Buffer{}
	go func() {
		defer wg.Done()
		if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	log.Error("scooby")
	r.Close()
	wg.Wait()

	if !strings.Contains(buf.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

}

func TestNewPipeReaderLevel(t *testing.T) {
	SetupLogging(Config{
		Level:  LevelDebug,
		Format: PlaintextOutput,
	})

	log := getLogger("test")

	var wg sync.WaitGroup
	wg.Add(1)

	r := NewPipeReader(PipeLevel(LevelError))

	buf := &bytes.Buffer{}
	go func() {
		defer wg.Done()
		if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	log.Debug("scooby")
	log.Info("velma")
	log.Error("shaggy")
	r.Close()
	wg.Wait()

	lineEnding := zap.NewProductionEncoderConfig().LineEnding

	// Should only contain one log line
	if strings.Count(buf.String(), lineEnding) > 1 {
		t.Errorf("got %d log lines, wanted 1", strings.Count(buf.String(), lineEnding))
	}

	if !strings.Contains(buf.String(), "shaggy") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

}
