package roundrobin

import (
	"testing"

	"github.com/micro/go-micro/v3/router"
	"github.com/micro/go-micro/v3/selector"
	"github.com/stretchr/testify/assert"
)

func TestRoundRobin(t *testing.T) {
	selector.Tests(t, NewSelector())

	r1 := router.Route{Service: "go.micro.service.foo", Address: "127.0.0.1:8000"}
	r2 := router.Route{Service: "go.micro.service.foo", Address: "127.0.0.1:8001"}
	r3 := router.Route{Service: "go.micro.service.foo", Address: "127.0.0.1:8002"}

	sel := NewSelector()

	// By passing r1 and r2 first, it forces a set sequence of (r1 => r2 => r3 => r1)

	r, err := sel.Select([]router.Route{r1})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r1, *r, "Expected route to be r1")

	r, err = sel.Select([]router.Route{r2})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r2, *r, "Expected route to be r2")

	// Because r1 and r2 have been recently called, r3 should be chosen

	r, err = sel.Select([]router.Route{r1, r2, r3})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r3, *r, "Expected route to be r3")

	// r1 was called longest ago, so it should be prioritised

	r, err = sel.Select([]router.Route{r1, r2, r3})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r1, *r, "Expected route to be r1")

	r, err = sel.Select([]router.Route{r1, r2, r3})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r2, *r, "Expected route to be r2")

	r, err = sel.Select([]router.Route{r1, r2, r3})
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, r3, *r, "Expected route to be r3")
}
