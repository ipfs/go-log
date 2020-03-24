package log

import (
	"sync"
	"testing"
)

// To run bencharks:
//   > go test -c .
//   > ./go-log.test -test.run NONE -test.bench . 2>/dev/null
// Otherwise you test how fast your terminal can print.

func BenchmarkSimpleInfo(b *testing.B) {
	l := Logger("bench")
	err := SetLogLevel("bench", "info")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Info("test")
	}
}

var logString = "String, IDK what to write, let's punch a keyboard. jkdlsjklfdjfklsjfklsdjaflkdjfkdjsfkldjsfkdjklfjdslfjakdfjioerjieofjofdnvonoijdfneslkffjsdfljadljfdjkfjkf"

func BenchmarkFormatInfo(b *testing.B) {
	l := Logger("bench")
	err := SetLogLevel("bench", "info")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Infof("test %d %s", logString)
	}
}

func BenchmarkFormatInfoMulti(b *testing.B) {
	l := Logger("bench")
	err := SetLogLevel("bench", "info")
	if err != nil {
		b.Fatal(err)
	}
	var wg sync.WaitGroup

	goroutines := 16

	run := func() {
		for i := 0; i < b.N/goroutines; i++ {
			l.Infof("test %d %s", i, logString)
		}
		wg.Done()
	}

	wg.Add(goroutines)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < goroutines; i++ {
		go run()
	}
	wg.Wait()
}
