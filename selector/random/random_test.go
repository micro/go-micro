package random

import (
	"testing"

	"github.com/asim/nitro/v3/selector"
)

func TestRandom(t *testing.T) {
	selector.Tests(t, NewSelector())
}
