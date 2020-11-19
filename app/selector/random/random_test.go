package random

import (
	"testing"

	"github.com/asim/nitro/v3/app/selector"
)

func TestRandom(t *testing.T) {
	selector.Tests(t, NewSelector())
}
