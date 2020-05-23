package log

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGetLoggerDefault(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// Call SetupLogging again so it picks up stderr change
	SetupLogging(Config{Stderr: true})
	log := getLogger("test")

	log.Error("scooby")
	w.Close()

	buf := &bytes.Buffer{}
	io.Copy(buf, r)

	if !strings.Contains(buf.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

}
