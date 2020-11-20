package random

import (
	"testing"

	"github.com/asim/nitro/app/selector"
)

func TestRandom(t *testing.T) {
	selector.Tests(t, NewSelector())
}
