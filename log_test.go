package log

import (
	"fmt"
	"testing"
)

// TestLogger_AnnotateName ensures that logger's system names are
// accurately annotated.
func TestLogger_AnnotateName(t *testing.T) {
	// set up several loggers
	Logger("test1")
	Logger("test2")
	Logger("test3")

	annotation := "annotation"

	for _, sub := range GetSubsystems() {
		Logger(sub).AnnotateName(annotation, true)
	}

	for _, sub := range GetSubsystems() {
		if sub[:len(annotation)] != annotation {
			t.Fatalf("expected %s, got %s", fmt.Sprintf("%s:%s", annotation, sub), sub)
		}
	}
}
