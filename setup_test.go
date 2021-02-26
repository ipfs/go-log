package log

import (
	"bytes"
	"io"
	"io/ioutil"
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
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "scooby") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

}

func TestLogToFileAndStderr(t *testing.T) {
	// setup stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// setup file
	logfile, err := ioutil.TempFile("", "go-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(logfile.Name())

	os.Setenv(envLoggingFile, logfile.Name())

	// set log output env var
	os.Setenv(envLoggingOutput, "file+stderr")

	SetupLogging(configFromEnv())

	log := getLogger("test")

	want := "scooby"
	log.Error(want)
	w.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), want) {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}

	content, err := ioutil.ReadFile(logfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), want) {
		t.Logf("want: '%s', got: '%s'", want, string(content))
		t.Fail()
	}
}

func TestLogToFile(t *testing.T) {
	// get tmp log file
	logfile, err := ioutil.TempFile("", "go-log-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(logfile.Name())

	// set the go-log file env var
	os.Setenv(envLoggingFile, logfile.Name())

	SetupLogging(configFromEnv())

	log := getLogger("test")

	// write log to file
	want := "grokgrokgrok"
	log.Error(want)

	// read log file and check contents
	content, err := ioutil.ReadFile(logfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), want) {
		t.Logf("want: '%s', got: '%s'", want, string(content))
		t.Fail()
	}
}

func TestLogLabels(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to open pipe: %v", err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	// set the go-log labels env var
	os.Setenv(envLoggingLabels, "app=example_app,dc=sjc-1")
	SetupLogging(configFromEnv())

	log := getLogger("test")

	log.Error("scooby")
	w.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Log(buf.String())
	if !strings.Contains(buf.String(), "{\"app\": \"example_app\", \"dc\": \"sjc-1\"}") {
		t.Errorf("got %q, wanted it to contain log output", buf.String())
	}
}
