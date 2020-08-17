package selector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests runs all the tests against a selector to ensure the implementations are consistent
func Tests(t *testing.T, s Selector) {
	r1 := "127.0.0.1:8000"
	r2 := "127.0.0.1:8001"

	t.Run("Select", func(t *testing.T) {
		t.Run("NoRoutes", func(t *testing.T) {
			_, err := s.Select([]string{})
			assert.Equal(t, ErrNoneAvailable, err, "Expected error to be none available")
		})

		t.Run("OneRoute", func(t *testing.T) {
			next, err := s.Select([]string{r1})
			srv := next()
			assert.Nil(t, err, "Error should be nil")
			assert.Equal(t, r1, srv, "Expected the route to be returned")
		})

		t.Run("MultipleRoutes", func(t *testing.T) {
			next, err := s.Select([]string{r1, r2})
			assert.Nil(t, err, "Error should be nil")
			srv := next()
			if srv != r1 && srv != r2 {
				t.Errorf("Expected the route to be one of the inputs")
			}
		})
	})

	t.Run("Record", func(t *testing.T) {
		err := s.Record(r1, nil)
		assert.Nil(t, err, "Expected the error to be nil")
	})

	t.Run("String", func(t *testing.T) {
		assert.NotEmpty(t, s.String(), "String returned a blank string")
	})
}
