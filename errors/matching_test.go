package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestStructuredErrorRoundTripMatching(t *testing.T) {
	original := &Error{Id: "media", Code: 412, Detail: "not processed", Reason: "MEDIA_NOT_READY", Domain: "media"}
	for _, format := range []string{"json", "proto"} {
		var remote Error
		if format == "json" {
			b, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(b, &remote); err != nil {
				t.Fatal(err)
			}
		} else {
			b, err := proto.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			if err = proto.Unmarshal(b, &remote); err != nil {
				t.Fatal(err)
			}
		}
		wrapped := fmt.Errorf("remote: %w", &remote)
		if !stderrors.Is(wrapped, FailedPrecondition("", "")) || !stderrors.Is(wrapped, &Error{Code: 412, Reason: "MEDIA_NOT_READY", Domain: "media"}) {
			t.Fatalf("%s failed to match", format)
		}
		if stderrors.Is(wrapped, &Error{Code: 412, Reason: "OTHER"}) || stderrors.Is(wrapped, NotFound("", "")) {
			t.Fatal("matched wrong error")
		}
		if got := FromError(wrapped); got != &remote {
			t.Fatal("wrapped structured error lost")
		}
		var target *Error
		if !stderrors.As(wrapped, &target) || target.Reason != original.Reason {
			t.Fatal("As failed")
		}
	}
}

func TestAdditionalErrorConstructors(t *testing.T) {
	for _, err := range []error{AlreadyExists("", "exists"), FailedPrecondition("", "precondition"), ResourceExhausted("", "limit"), Unavailable("", "offline")} {
		structured, ok := As(err)
		if !ok || structured.Status == "" {
			t.Fatalf("invalid error: %v", err)
		}
	}
}
