package acme

import (
	"testing"
)

func TestDefault(t *testing.T) {
	a := Default()
	switch b := a.(type) {
	case *AutoCert:
	default:
		t.Fatalf("Expected Default to return *AutoCert, got %T", b)
	}
}

func TestNew(t *testing.T) {
	for _, x := range []string{LibAutoCert} {
		_, err := New(x)
		if err != nil {
			t.Error(err.Error())
		}
	}

	_, err := New("foo")
	if err != ErrUnsupportedLibrary {
		t.Error("New() returned an error that was not ErrUnsupportedLibrary")
	}
}

func TestAutoCert(t *testing.T) {
	var a Library
	a = &AutoCert{}

	_, err := a.NewListener([]string{"test.localhost"})
	if err != nil {
		t.Fatal("AutoCert.NewListener failed")
	}
}
