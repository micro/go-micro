package agent

import (
	"strings"
	"testing"
)

// Repeating the same tool call with the same arguments is refused once it
// exceeds LoopLimit, and the model is told to change approach.
func TestLoopDetectionStopsRepeats(t *testing.T) {
	a := newTestAgent(Name("looper"), LoopLimit(3))
	h := a.toolHandler()

	// First 3 identical calls are allowed (they fall through to RPC,
	// which fails harmlessly — we only care they weren't refused as loops).
	for i := 1; i <= 3; i++ {
		content := toolContent(h, "demo_Svc_Do", map[string]any{"q": "x"})
		if strings.Contains(content, "loop detected") {
			t.Fatalf("call %d wrongly flagged as a loop", i)
		}
	}

	// The 4th identical call is refused as a loop.
	content := toolContent(h, "demo_Svc_Do", map[string]any{"q": "x"})
	if !strings.Contains(content, "loop detected") {
		t.Errorf("4th identical call should be refused as a loop; got %q", content)
	}
}

// Different arguments are not a loop, even past the limit.
func TestLoopDetectionAllowsDistinctCalls(t *testing.T) {
	a := newTestAgent(Name("distinct"), LoopLimit(2))
	h := a.toolHandler()

	for i := 0; i < 5; i++ {
		content := toolContent(h, "demo_Svc_Do", map[string]any{"q": i}) // distinct args each time
		if strings.Contains(content, "loop detected") {
			t.Fatalf("distinct call %d wrongly flagged as a loop", i)
		}
	}
}

// LoopLimit(0) disables detection.
func TestLoopDetectionDisabled(t *testing.T) {
	a := newTestAgent(Name("noloop"), LoopLimit(0))
	h := a.toolHandler()
	for i := 0; i < 6; i++ {
		content := toolContent(h, "demo_Svc_Do", map[string]any{"q": "same"})
		if strings.Contains(content, "loop detected") {
			t.Fatalf("loop detection should be disabled with LoopLimit(0)")
		}
	}
}

// It defaults on (lenient) so repeated identical calls are caught without
// any configuration.
func TestLoopDetectionDefaultOn(t *testing.T) {
	a := New(Name("d"), Provider("fake")).(*agentImpl)
	a.setup()
	if a.opts.LoopLimit <= 0 {
		t.Fatalf("LoopLimit should default on, got %d", a.opts.LoopLimit)
	}
	h := a.toolHandler()
	var lastContent string
	for i := 0; i < a.opts.LoopLimit+1; i++ {
		lastContent = toolContent(h, "demo_Svc_Do", map[string]any{})
	}
	if !strings.Contains(lastContent, "loop detected") {
		t.Errorf("default loop detection should catch repeated calls; got %q", lastContent)
	}
}
