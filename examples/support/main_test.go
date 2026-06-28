package main

import "testing"

func TestRunSupportMockSmoke(t *testing.T) {
	if err := runSupport("mock"); err != nil {
		t.Fatalf("support example failed: %v", err)
	}
}
