package autocert

import (
	"testing"
)

func TestAutocert(t *testing.T) {
	l := New()
	if _, ok := l.(*autocertACME); !ok {
		t.Error("New() didn't return an autocertACME")
	}
	if _, err := l.NewListener(); err != nil {
		t.Error(err.Error())
	}
}
