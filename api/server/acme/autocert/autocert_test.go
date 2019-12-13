package autocert

import (
	"testing"
)

func TestAutocert(t *testing.T) {
	l := New()
	if _, ok := l.(*autocertProvider); !ok {
		t.Error("New() didn't return an autocertProvider")
	}
	// TODO: Travis CI doesn't let us bind :443
	// if _, err := l.NewListener(); err != nil {
	// 	t.Error(err.Error())
	// }
}
