package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {

	// Make some new options:
	options := NewOptions(
		Address(":9999"),
		DefaultTags(map[string]string{"service": "prometheus-test"}),
		Path("/prometheus"),
		Percentiles([]float64{0.11, 0.22, 0.33}),
	)

	// Check that the defaults and overrides were accepted:
	assert.Equal(t, ":9999", options.Address)
	assert.Equal(t, "prometheus-test", options.DefaultTags["service"])
	assert.Equal(t, "/prometheus", options.Path)
	assert.Equal(t, []float64{0.11, 0.22, 0.33}, options.Percentiles)
}
