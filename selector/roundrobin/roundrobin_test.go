package roundrobin

import (
	"testing"

	"github.com/micro/go-micro/v3/selector"
	"github.com/stretchr/testify/assert"
)

func TestRoundRobin(t *testing.T) {
	selector.Tests(t, NewSelector())

	r1 := "127.0.0.1:8000"
	r2 := "127.0.0.1:8001"
	r3 := "127.0.0.1:8002"

	sel := NewSelector()

	// By passing r1 and r2 first, it forces a set sequence of (r1 => r2 => r3 => r1)

	next, err := sel.Select([]string{r1})
	r := next()
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r1, r, "Expected route to be r1")

	next, err = sel.Select([]string{r2})
	r = next()
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r2, r, "Expected route to be r2")

	// Because r1 and r2 have been recently called, r3 should be chosen

	next, err = sel.Select([]string{r1, r2, r3})
	n1, n2, n3 := next(), next(), next()
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r1, n1, "Expected route to be r3")
	assert.Equal(t, r2, n2, "Expected route to be r3")
	assert.Equal(t, r3, n3, "Expected route to be r3")

}
