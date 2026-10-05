package main

import "testing"

// getVersion must always yield something printable: an injected, stamped,
// or probed revision, else the dev fallback. Never empty.
func TestGetVersionNonEmpty(t *testing.T) {
	if v := getVersion(); v == "" {
		t.Fatal("getVersion returned empty string")
	}
}

// An ldflags-injected version wins over every fallback.
func TestGetVersionInjected(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = "v9.9.9-test"
	if v := getVersion(); v != "v9.9.9-test" {
		t.Fatalf("getVersion = %q, want injected version", v)
	}
}
