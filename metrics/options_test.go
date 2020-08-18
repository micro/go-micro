package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {

	// Make some new options:
	options := NewOptions(Path("/prometheus"), DefaultTags(map[string]string{"service": "prometheus-test"}))

	// Check that the defaults and overrides were accepted:
	assert.Equal(t, ":9000", options.Address)
	assert.Equal(t, "/prometheus", options.Path)
	assert.Equal(t, "prometheus-test", options.DefaultTags["service"])
}
