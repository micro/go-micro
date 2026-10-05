package main

import "testing"

// getVersion must always yield something printable: an injected, stamped,
// or probed revision, else the dev fallback. Never empty.
func TestGetVersionNonEmpty(t *testing.T) {
	if v := getVersion(); v == "" {
		t.Fatal("getVersion returned empty string")
	}
}
