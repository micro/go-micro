package harnessutil

import (
	"testing"
	"time"
)

func TestLiveTimeoutLeavesMockUnchanged(t *testing.T) {
	t.Setenv(LiveTimeoutEnv, "2m")
	if got := LiveTimeout("mock"); got != 0 {
		t.Fatalf("LiveTimeout(mock) = %s, want 0", got)
	}
	if opts := AgentOptions("mock"); len(opts) != 0 {
		t.Fatalf("AgentOptions(mock) length = %d, want 0", len(opts))
	}
}

func TestLiveTimeoutUsesDefaultForLiveProviders(t *testing.T) {
	t.Setenv(LiveTimeoutEnv, "")
	if got := LiveTimeout("atlascloud"); got != DefaultLiveTimeout {
		t.Fatalf("LiveTimeout(live) = %s, want %s", got, DefaultLiveTimeout)
	}
	if opts := AgentOptions("atlascloud"); len(opts) != 2 {
		t.Fatalf("AgentOptions(live) length = %d, want 2", len(opts))
	}
}

func TestLiveTimeoutCanBeOverridden(t *testing.T) {
	t.Setenv(LiveTimeoutEnv, "90s")
	if got := LiveTimeout("anthropic"); got != 90*time.Second {
		t.Fatalf("LiveTimeout override = %s, want 90s", got)
	}
}
