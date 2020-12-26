package url

import (
	"testing"
)

func TestFormat(t *testing.T) {
	testCases := []struct {
		contentType string
		format      string
	}{
		{"application/json", "json"},
		{"application/xml", "xml"},
		{"application/json", "json"},
	}

	for _, c := range testCases {
		f := format(c.contentType)
		if f != c.format {
			t.Fatalf("failed to format %s: expected %s got %s", c.contentType, c.format, f)
		}
	}
}
