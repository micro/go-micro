package memory

import (
	"reflect"
	"testing"

	"github.com/micro/go-micro/v2/debug/log"
)

func TestLogger(t *testing.T) {
	// set size to some value
	size := 100
	// override the global logger
	lg := NewLog(log.Size(size))
	// make sure we have the right size of the logger ring buffer
	if lg.(*memoryLog).Size() != size {
		t.Errorf("expected buffer size: %d, got: %d", size, lg.(*memoryLog).Size())
	}

	// Log some cruft
	lg.Write(log.Record{Message: "foobar"})
	lg.Write(log.Record{Message: "foo bar"})

	// Check if the logs are stored in the logger ring buffer
	expected := []string{"foobar", "foo bar"}
	entries, _ := lg.Read(log.Count(len(expected)))
	for i, entry := range entries {
		if !reflect.DeepEqual(entry.Message, expected[i]) {
			t.Errorf("expected %s, got %s", expected[i], entry.Message)
		}
	}
}
