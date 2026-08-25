package profile

import (
	"strings"
	"testing"
)

// A profile that is not linked into the binary must fail with an error that
// names the import which registers it — not a bare "unsupported profile".
func TestLoadUnregisteredProfileNamesTheImport(t *testing.T) {
	_, err := Load("nats")
	if err == nil {
		t.Fatal("expected error for unregistered profile in this package's own test binary")
	}
	if !strings.Contains(err.Error(), "cmd/defaults") {
		t.Errorf("error %q should point at go-micro.dev/v6/cmd/defaults", err)
	}
}

func TestLocalProfileRegistered(t *testing.T) {
	if _, err := Load("local"); err != nil {
		t.Fatalf("local profile must always be loadable: %v", err)
	}
}
