package log

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
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
		if _, err := io.Copy(buf, r); err != nil {
			require.ErrorIs(t, err, io.ErrClosedPipe)
		}
	}()

	log.Error("scooby")
	r.Close()
	wg.Wait()

	require.Contains(t, buf.String(), "scooby")
}

func TestNewPipeReaderFormat(t *testing.T) {
	log := getLogger("test")

	var wg sync.WaitGroup
	wg.Add(1)

	r := NewPipeReader(PipeFormat(PlaintextOutput))

	buf := &bytes.Buffer{}
	go func() {
		defer wg.Done()
		if _, err := io.Copy(buf, r); err != nil {
			require.ErrorIs(t, err, io.ErrClosedPipe)
		}
	}()

	log.Error("scooby")
	r.Close()
	wg.Wait()

	require.Contains(t, buf.String(), "scooby")
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
		if _, err := io.Copy(buf, r); err != nil {
			require.ErrorIs(t, err, io.ErrClosedPipe)
		}
	}()

	log.Debug("scooby")
	log.Info("velma")
	log.Error("shaggy")
	r.Close()
	wg.Wait()

	lineEnding := zap.NewProductionEncoderConfig().LineEnding

	// Should only contain one log line
	require.Equal(t, 1, strings.Count(buf.String(), lineEnding), "expected 1 log line")
	require.Contains(t, buf.String(), "shaggy")
}
