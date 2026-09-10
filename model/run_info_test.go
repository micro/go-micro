package model

import (
	"context"
	"strings"
	"testing"

	"go-micro.dev/v6/metadata"
)

func TestRunInfoCrossesMetadataBoundary(t *testing.T) {
	want := RunInfo{
		RunID:    "flow-run-1",
		ParentID: "parent-run-1",
		Agent:    "checkout",
		Flow:     "checkout",
		Step:     "authorize",
		Dispatch: "schedule",
		Trigger:  "daily-settlement",
	}
	origin := WithRunInfo(context.Background(), want)
	md, ok := metadata.FromContext(origin)
	if !ok {
		t.Fatal("WithRunInfo did not attach RPC metadata")
	}
	for key := range md {
		if key != strings.ToLower(key) {
			t.Fatalf("metadata key %q is not gRPC-normalized", key)
		}
	}

	// Rebuild the server-side context from metadata only, as an RPC
	// transport does. The process-local context value is intentionally gone.
	server := metadata.NewContext(context.Background(), md)
	got, ok := RunInfoFrom(server)
	if !ok {
		t.Fatal("RunInfoFrom did not recover transported metadata")
	}
	if got != want {
		t.Fatalf("RunInfo = %#v, want %#v", got, want)
	}
}

func TestRunInfoLocalValueTakesPrecedenceOverMetadata(t *testing.T) {
	ctx := WithRunInfo(context.Background(), RunInfo{RunID: "local-run", Attempt: 2})
	ctx = metadata.Set(ctx, runHeaderID, "transport-run")
	got, ok := RunInfoFrom(ctx)
	if !ok {
		t.Fatal("RunInfoFrom returned no run info")
	}
	if got.RunID != "local-run" || got.Attempt != 2 {
		t.Fatalf("RunInfo = %#v, want local value", got)
	}
}
