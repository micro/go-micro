package main

import "testing"

func TestRunFirstAgent(t *testing.T) {
	if err := runFirstAgent(); err != nil {
		t.Fatalf("first-agent example failed: %v", err)
	}
}
