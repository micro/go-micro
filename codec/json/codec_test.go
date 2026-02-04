package json

import (
	"bytes"
	"encoding/json"
	"testing"

	"go-micro.dev/v5/codec"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// mockReadWriteCloser implements io.ReadWriteCloser for testing
type mockReadWriteCloser struct {
	*bytes.Buffer
}

func (m *mockReadWriteCloser) Close() error {
	return nil
}

// TestCodecAnyTypeWrite tests that google.protobuf.Any types are properly written with @type field
func TestCodecAnyTypeWrite(t *testing.T) {
	buf := &mockReadWriteCloser{Buffer: bytes.NewBuffer(nil)}
	c := NewCodec(buf).(*Codec)

	// Create a StringValue message
	stringValue := wrapperspb.String("test value")

	// Wrap it in an Any message
	anyMsg, err := anypb.New(stringValue)
	if err != nil {
		t.Fatalf("Failed to create Any message: %v", err)
	}

	// Write the message
	msg := &codec.Message{
		Type: codec.Response,
	}
	if err := c.Write(msg, anyMsg); err != nil {
		t.Fatalf("Failed to write Any message: %v", err)
	}

	// Parse the written JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that @type field exists
	typeURL, ok := result["@type"].(string)
	if !ok {
		t.Fatalf("@type field not found in JSON output. Got: %v", buf.String())
	}

	// Verify the type URL is correct
	expectedTypeURL := "type.googleapis.com/google.protobuf.StringValue"
	if typeURL != expectedTypeURL {
		t.Errorf("Expected @type to be %s, got %s", expectedTypeURL, typeURL)
	}

	t.Logf("Successfully wrote Any type with @type field: %s", buf.String())
}

// TestCodecAnyTypeRead tests that JSON with @type field can be read into google.protobuf.Any
func TestCodecAnyTypeRead(t *testing.T) {
	// JSON representation of an Any message with @type field
	jsonData := `{"@type":"type.googleapis.com/google.protobuf.StringValue","value":"test value"}`

	buf := &mockReadWriteCloser{Buffer: bytes.NewBufferString(jsonData + "\n")}
	c := NewCodec(buf).(*Codec)

	// Read into an Any message
	anyMsg := &anypb.Any{}
	if err := c.ReadBody(anyMsg); err != nil {
		t.Fatalf("Failed to read Any message: %v", err)
	}

	// Verify the type URL is set
	expectedTypeURL := "type.googleapis.com/google.protobuf.StringValue"
	if anyMsg.TypeUrl != expectedTypeURL {
		t.Errorf("Expected TypeUrl to be %s, got %s", expectedTypeURL, anyMsg.TypeUrl)
	}

	// Unmarshal the contained message
	stringValue := &wrapperspb.StringValue{}
	if err := anyMsg.UnmarshalTo(stringValue); err != nil {
		t.Fatalf("Failed to unmarshal contained message: %v", err)
	}

	// Verify the value
	expectedValue := "test value"
	if stringValue.Value != expectedValue {
		t.Errorf("Expected value to be %s, got %s", expectedValue, stringValue.Value)
	}

	t.Logf("Successfully read Any type from JSON with @type field")
}
