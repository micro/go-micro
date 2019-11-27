package log

import (
	"reflect"
	"testing"
)

func TestLogger(t *testing.T) {
	// set size to some value
	size := 100
	// override the global logger
	logger = NewLog(Size(size))
	// make sure we have the right size of the logger ring buffer
	if logger.(*defaultLog).Size() != size {
		t.Errorf("expected buffer size: %d, got: %d", size, logger.(*defaultLog).Size())
	}

	// Log some cruft
	Info("foobar")
	// increase the log level
	level = LevelDebug
	Debugf("foo %s", "bar")

	// Check if the logs are stored in the logger ring buffer
	expected := []string{"foobar", "foo bar"}
	entries := logger.Read(Count(len(expected)))
	for i, entry := range entries {
		if !reflect.DeepEqual(entry.Value, expected[i]) {
			t.Errorf("expected %s, got %s", expected[i], entry.Value)
		}
	}
}
