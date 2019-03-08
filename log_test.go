package log

import (
	"encoding/json"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type jsonLog struct {
	File   string
	Func   string
	Level  string
	Msg    string
	System string
	Time   string
	Field  string
	Humpty string
}

func TestLogSimple(t *testing.T) {
	assert := assert.New(t)
	log := Logger("testing")
	r, w := io.Pipe()

	SetAllLogFormatJSON()
	SetLogOutput("testing", w)

	var wg sync.WaitGroup
	decoder := json.NewDecoder(r)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var msg jsonLog
		if err := decoder.Decode(&msg); err != nil {
			t.Fatal(err)
		}
		assert.Equal("testing", msg.System)
		assert.Equal("info", msg.Level)
		assert.Equal("bar", msg.Field)
		assert.Equal("Dumpty", msg.Humpty)
	}()

	log.WithFields(Fields{"Field": "bar", "Humpty": "Dumpty"}).Info("test")
	wg.Wait()
}
