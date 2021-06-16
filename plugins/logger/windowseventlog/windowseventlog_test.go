package windowseventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {

	l := NewLogger()

	assert.Equal(t, l.String(), "windowseventlog")

}
