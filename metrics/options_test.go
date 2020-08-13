package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {

	// Make some new options:
	options := NewOptions(PrometheusPath("/prometheus"), DefaultTags(map[string]string{"service": "prometheus-test"}))

	// Check that the defaults and overrides were accepted:
	assert.Equal(t, "prometheus-test", options.DefaultTags["service"])
	assert.Equal(t, ":9000", options.PrometheusListenAddress)
	assert.Equal(t, "/prometheus", options.PrometheusPath)
}
