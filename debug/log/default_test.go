package log

import (
	"reflect"
	"testing"
)

func TestLogger(t *testing.T) {
	// make sure we have the right size of the logger ring buffer
	if logger.(*defaultLogger).Size() != DefaultSize {
		t.Errorf("expected buffer size: %d, got: %d", DefaultSize, logger.(*defaultLogger).Size())
	}

	// Log some cruft
	Log("foobar")
	Logf("foo %s", "bar")

	// Check if the logs are stored in the logger ring buffer
	expectedEntries := []string{"foobar", "foo bar"}
	entries := logger.Read(len(expectedEntries))
	for i, entry := range entries {
		if !reflect.DeepEqual(entry, expectedEntries[i]) {
			t.Errorf("expected %s, got %s", expectedEntries[i], entry)
		}
	}
}
