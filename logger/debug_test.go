package logger

import (
	"testing"

	dlog "go-micro.dev/v5/debug/log"
)

func TestDebugLogBuffer(t *testing.T) {
	// Create a new logger
	l := NewLogger(WithLevel(InfoLevel))

	// Log some messages
	l.Log(InfoLevel, "test message 1")
	l.Log(WarnLevel, "test message 2")
	l.Logf(ErrorLevel, "formatted message %d", 3)

	// Read from debug log buffer
	records, err := dlog.DefaultLog.Read()
	if err != nil {
		t.Fatalf("Failed to read from debug log: %v", err)
	}

	// We should have at least our 3 messages
	if len(records) < 3 {
		t.Fatalf("Expected at least 3 log records in debug buffer, got %d", len(records))
	}

	// Check that our messages are there
	foundCount := 0
	for _, rec := range records {
		msg, ok := rec.Message.(string)
		if !ok {
			continue
		}
		if msg == "test message 1" || msg == "test message 2" || msg == "formatted message 3" {
			foundCount++
			// Verify metadata is present
			if rec.Metadata == nil {
				t.Errorf("Record has nil metadata")
			}
			// Verify level is in metadata
			if _, ok := rec.Metadata["level"]; !ok {
				t.Errorf("Record missing level in metadata")
			}
		}
	}

	if foundCount < 3 {
		t.Errorf("Expected to find 3 specific messages in debug log, found %d", foundCount)
	}
}

func TestDebugLogWithFields(t *testing.T) {
	// Create a logger with fields
	l := NewLogger(WithLevel(InfoLevel), WithFields(map[string]interface{}{
		"service": "test",
		"version": "1.0",
	}))

	// Log a message
	l.Log(InfoLevel, "message with fields")

	// Read from debug log buffer
	records, err := dlog.DefaultLog.Read()
	if err != nil {
		t.Fatalf("Failed to read from debug log: %v", err)
	}

	// Find our message
	found := false
	for _, rec := range records {
		msg, ok := rec.Message.(string)
		if !ok {
			continue
		}
		if msg == "message with fields" {
			found = true
			// Verify fields are in metadata
			if rec.Metadata["service"] != "test" {
				t.Errorf("Expected service=test in metadata, got %s", rec.Metadata["service"])
			}
			if rec.Metadata["version"] != "1.0" {
				t.Errorf("Expected version=1.0 in metadata, got %s", rec.Metadata["version"])
			}
			break
		}
	}

	if !found {
		t.Error("Did not find message with fields in debug log")
	}
}
