package random

import (
	"testing"

	"github.com/asim/go-micro/v3/selector"
)

func TestRandom(t *testing.T) {
	selector.Tests(t, NewSelector())
}
