package grpc

import (
	"go-micro.dev/v6/errors"
	"google.golang.org/grpc/codes"
	"testing"
)

func TestAdditionalErrorStatusMapping(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code codes.Code
	}{{errors.AlreadyExists("", ""), codes.AlreadyExists}, {errors.FailedPrecondition("", ""), codes.FailedPrecondition}, {errors.ResourceExhausted("", ""), codes.ResourceExhausted}, {errors.Unavailable("", ""), codes.Unavailable}} {
		if got := microError(errors.FromError(tc.err)); got != tc.code {
			t.Fatalf("%v -> %v, want %v", tc.err, got, tc.code)
		}
	}
}
