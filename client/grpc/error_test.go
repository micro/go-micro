package grpc

import (
	stderrors "errors"
	"testing"

	"go-micro.dev/v6/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStructuredStatusDetails(t *testing.T) {
	original := &errors.Error{Code: 429, Reason: "QUOTA", Domain: "billing"}
	st, err := status.New(codes.ResourceExhausted, "quota").WithDetails(&errdetails.ErrorInfo{Reason: "ignored"}, original)
	if err != nil {
		t.Fatal(err)
	}
	got := microError(st.Err())
	if !stderrors.Is(got, original) {
		t.Fatalf("lost error details: %v", got)
	}
	st, err = status.New(codes.ResourceExhausted, "quota").WithDetails(&errdetails.ErrorInfo{Reason: "quota"})
	if err != nil {
		t.Fatal(err)
	}
	if !stderrors.Is(microError(st.Err()), errors.ResourceExhausted("", "")) {
		t.Fatal("429 fallback mapping failed")
	}
}
