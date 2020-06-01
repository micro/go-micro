package random

import (
	"testing"

	"github.com/micro/go-micro/v2/selector"
)

func TestRandom(t *testing.T) {
	selector.Tests(t, NewSelector())
}
