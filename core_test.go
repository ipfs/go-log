package log

import (
	"bytes"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestNewCoreFormat(t *testing.T) {
	entry := zapcore.Entry{
		LoggerName: "main",
		Level:      zapcore.InfoLevel,
		Message:    "scooby",
		Time:       time.Date(2010, 5, 23, 15, 14, 0, 0, time.UTC),
	}

	testCases := []struct {
		format LogFormat
		want   string
	}{
		{
			format: ColorizedOutput,
			want:   "2010-05-23T15:14:00.000Z\t\x1b[34mINFO\x1b[0m\tmain\tscooby\n",
		},
		{
			format: JSONOutput,
			want:   `{"level":"info","ts":"2010-05-23T15:14:00.000Z","logger":"main","msg":"scooby"}` + "\n",
		},
		{
			format: PlaintextOutput,
			want:   "2010-05-23T15:14:00.000Z\tINFO\tmain\tscooby\n",
		},
	}

	for _, tc := range testCases {
		buf := &bytes.Buffer{}
		ws := zapcore.AddSync(buf)

		core := newCore(tc.format, ws, LevelDebug)
		if err := core.Write(entry, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := buf.String()
		if got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}

}

func TestLockedMultiCoreAddCore(t *testing.T) {
	mc := &lockedMultiCore{}

	buf1 := &bytes.Buffer{}
	core1 := newCore(PlaintextOutput, zapcore.AddSync(buf1), LevelDebug)
	mc.AddCore(core1)

	buf2 := &bytes.Buffer{}
	core2 := newCore(ColorizedOutput, zapcore.AddSync(buf2), LevelDebug)
	mc.AddCore(core2)

	entry := zapcore.Entry{
		LoggerName: "main",
		Level:      zapcore.InfoLevel,
		Message:    "scooby",
		Time:       time.Date(2010, 5, 23, 15, 14, 0, 0, time.UTC),
	}
	if err := mc.Write(entry, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want1 := "2010-05-23T15:14:00.000Z\tINFO\tmain\tscooby\n"
	got1 := buf1.String()
	if got1 != want1 {
		t.Errorf("core1 got %q, want %q", got1, want1)
	}

	want2 := "2010-05-23T15:14:00.000Z\t\x1b[34mINFO\x1b[0m\tmain\tscooby\n"
	got2 := buf2.String()
	if got2 != want2 {
		t.Errorf("core2 got %q, want %q", got2, want2)
	}

}

func TestLockedMultiCoreDeleteCore(t *testing.T) {

	mc := &lockedMultiCore{}

	buf1 := &bytes.Buffer{}
	core1 := newCore(PlaintextOutput, zapcore.AddSync(buf1), LevelDebug)
	mc.AddCore(core1)

	// Write entry to just first core
	entry := zapcore.Entry{
		LoggerName: "main",
		Level:      zapcore.InfoLevel,
		Message:    "scooby",
		Time:       time.Date(2010, 5, 23, 15, 14, 0, 0, time.UTC),
	}
	if err := mc.Write(entry, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf2 := &bytes.Buffer{}
	core2 := newCore(ColorizedOutput, zapcore.AddSync(buf2), LevelDebug)
	mc.AddCore(core2)

	// Remove the first core
	mc.DeleteCore(core1)

	// Write another entry
	entry2 := zapcore.Entry{
		LoggerName: "main",
		Level:      zapcore.InfoLevel,
		Message:    "velma",
		Time:       time.Date(2010, 5, 23, 15, 15, 0, 0, time.UTC),
	}

	if err := mc.Write(entry2, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want1 := "2010-05-23T15:14:00.000Z\tINFO\tmain\tscooby\n"
	got1 := buf1.String()
	if got1 != want1 {
		t.Errorf("core1 got %q, want %q", got1, want1)
	}

	want2 := "2010-05-23T15:15:00.000Z\t\x1b[34mINFO\x1b[0m\tmain\tvelma\n"
	got2 := buf2.String()
	if got2 != want2 {
		t.Errorf("core2 got %q, want %q", got2, want2)
	}

}

func TestLockedMultiCoreReplaceCore(t *testing.T) {
	mc := &lockedMultiCore{}

	buf1 := &bytes.Buffer{}
	core1 := newCore(PlaintextOutput, zapcore.AddSync(buf1), LevelDebug)
	mc.AddCore(core1)

	// Write entry to just first core
	entry := zapcore.Entry{
		LoggerName: "main",
		Level:      zapcore.InfoLevel,
		Message:    "scooby",
		Time:       time.Date(2010, 5, 23, 15, 14, 0, 0, time.UTC),
	}
	if err := mc.Write(entry, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf2 := &bytes.Buffer{}
	core2 := newCore(ColorizedOutput, zapcore.AddSync(buf2), LevelDebug)

	// Replace the first core with the second
	mc.ReplaceCore(core1, core2)

	// Write another entry
	entry2 := zapcore.Entry{
		LoggerName: "main",
		Level:      zapcore.InfoLevel,
		Message:    "velma",
		Time:       time.Date(2010, 5, 23, 15, 15, 0, 0, time.UTC),
	}

	if err := mc.Write(entry2, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want1 := "2010-05-23T15:14:00.000Z\tINFO\tmain\tscooby\n"
	got1 := buf1.String()
	if got1 != want1 {
		t.Errorf("core1 got %q, want %q", got1, want1)
	}

	want2 := "2010-05-23T15:15:00.000Z\t\x1b[34mINFO\x1b[0m\tmain\tvelma\n"
	got2 := buf2.String()
	if got2 != want2 {
		t.Errorf("core2 got %q, want %q", got2, want2)
	}

}
