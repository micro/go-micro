package main

import (
	"strings"
	"testing"
)

func TestReadSSEDataAcceptsMultipleJSONEvents(t *testing.T) {
	payload, err := readSSEData(strings.NewReader("event: status\ndata: {\"phase\":\"started\"}\n\ndata:{\"result\":\"a2a-fallback-ok\"}\n\n"))
	if err != nil {
		t.Fatalf("readSSEData returned error: %v", err)
	}
	if !strings.Contains(payload, "started") || !strings.Contains(payload, "a2a-fallback-ok") {
		t.Fatalf("payload = %q, want both event payloads", payload)
	}
}

func TestReadSSEDataRejectsInvalidJSONEvent(t *testing.T) {
	_, err := readSSEData(strings.NewReader("data: {\"ok\":true}\n\ndata: {bad json}\n\n"))
	if err == nil {
		t.Fatal("readSSEData returned nil error for invalid JSON event")
	}
}
