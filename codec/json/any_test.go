package json

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestAnyTypeMarshaling tests that google.protobuf.Any types are properly marshaled with @type field
func TestAnyTypeMarshaling(t *testing.T) {
	marshaler := Marshaler{}

	// Create a StringValue message
	stringValue := wrapperspb.String("test value")

	// Wrap it in an Any message
	anyMsg, err := anypb.New(stringValue)
	if err != nil {
		t.Fatalf("Failed to create Any message: %v", err)
	}

	// Marshal using our JSON marshaler
	data, err := marshaler.Marshal(anyMsg)
	if err != nil {
		t.Fatalf("Failed to marshal Any message: %v", err)
	}

	// Unmarshal into a map to check for @type field
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that @type field exists
	typeURL, ok := result["@type"].(string)
	if !ok {
		t.Fatalf("@type field not found in JSON output. Got: %v", string(data))
	}

	// Verify the type URL is correct
	expectedTypeURL := "type.googleapis.com/google.protobuf.StringValue"
	if typeURL != expectedTypeURL {
		t.Errorf("Expected @type to be %s, got %s", expectedTypeURL, typeURL)
	}

	// Verify the value field exists
	if _, ok := result["value"]; !ok {
		t.Errorf("value field not found in JSON output. Got: %v", string(data))
	}

	t.Logf("Successfully marshaled Any type with @type field: %s", string(data))
}

// TestAnyTypeUnmarshaling tests that JSON with @type field can be unmarshaled into google.protobuf.Any
func TestAnyTypeUnmarshaling(t *testing.T) {
	marshaler := Marshaler{}

	// JSON representation of an Any message with @type field
	jsonData := []byte(`{
		"@type": "type.googleapis.com/google.protobuf.StringValue",
		"value": "test value"
	}`)

	// Unmarshal into an Any message
	anyMsg := &anypb.Any{}
	if err := marshaler.Unmarshal(jsonData, anyMsg); err != nil {
		t.Fatalf("Failed to unmarshal Any message: %v", err)
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

	t.Logf("Successfully unmarshaled Any type from JSON with @type field")
}
