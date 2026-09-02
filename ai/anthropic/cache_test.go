package anthropic

import (
	"encoding/json"
	"strings"
	"testing"
)

// A big prefix is marked for caching, once, at the end of the system prompt.
//
// An agent's request is mostly the same request every time: the tools and the
// system prompt do not change between turns, or between the rounds of one tool
// loop. Uncached that is the whole catalogue re-sent and re-billed on every
// call — for a caller with a hundred tools, tens of thousands of tokens a turn.
func TestABigPrefixIsMarkedForCaching(t *testing.T) {
	tools := []map[string]any{}
	for i := 0; i < 40; i++ {
		tools = append(tools, map[string]any{
			"name":        "service_method",
			"description": strings.Repeat("what this tool does and when to reach for it. ", 6),
		})
	}

	got := cacheableSystem("You are an assistant with tools.", tools, false)
	blocks, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("system is %T, so nothing is cached", got)
	}
	if len(blocks) != 1 {
		t.Fatalf("%d blocks; one breakpoint is wanted, not several", len(blocks))
	}
	if blocks[0]["cache_control"] == nil {
		t.Error("the block carries no cache_control, so the prefix is not cached")
	}
	if blocks[0]["text"] != "You are an assistant with tools." {
		t.Errorf("the system prompt did not survive: %v", blocks[0]["text"])
	}
	// It must still marshal to something the API accepts.
	if _, err := json.Marshal(map[string]any{"system": got}); err != nil {
		t.Fatalf("the request will not marshal: %v", err)
	}
}

// The breakpoint goes on the system prompt and not on the tools, because a
// request is ordered tools, then system, then messages — so one mark at the end
// of the system prompt caches both, and marking the tools as well would spend a
// second of the four breakpoints a request is allowed for nothing.
func TestTheToolsAreNotMarkedSeparately(t *testing.T) {
	tools := []map[string]any{{"name": "a", "description": strings.Repeat("x", 5000)}}
	cacheableSystem("system", tools, false)
	for _, tl := range tools {
		if _, marked := tl["cache_control"]; marked {
			t.Error("a tool was marked; the system breakpoint already covers them")
		}
	}
}

// Below the smallest cacheable prefix it stays a plain string. Asking the API
// to cache less than it will cache is an error, and the thing being optimised
// is not present anyway.
func TestASmallRequestIsLeftAlone(t *testing.T) {
	for _, tc := range []struct {
		name   string
		system string
		tools  []map[string]any
	}{
		{"no tools, short prompt", "Be helpful.", nil},
		{"empty prompt", "", nil},
		{"whitespace only", "   \n ", nil},
	} {
		got := cacheableSystem(tc.system, tc.tools, false)
		if _, isString := got.(string); !isString {
			t.Errorf("%s: system is %T, want a plain string", tc.name, got)
		}
	}
}

// Tools count towards the threshold, because they are inside the cached prefix.
// A one-line system prompt with a hundred tools behind it is well worth caching
// and would not reach the threshold on its own.
func TestToolsCountTowardsTheThreshold(t *testing.T) {
	short := "Be helpful."
	if _, plain := cacheableSystem(short, nil, false).(string); !plain {
		t.Fatal("a short prompt with no tools should be left alone")
	}
	big := []map[string]any{{"name": "a", "description": strings.Repeat("x", minCacheBytes)}}
	if _, plain := cacheableSystem(short, big, false).(string); plain {
		t.Error("the same short prompt with a big tool list was left uncached")
	}
}

// With no system prompt to carry the breakpoint, a large tool catalogue gets
// it on the last tool instead — otherwise the whole catalogue is re-sent and
// re-billed on every call of a system-prompt-less caller. The original slice
// and its maps are left untouched.
func TestAToolOnlyPrefixIsCachedOnTheLastTool(t *testing.T) {
	tools := []map[string]any{
		{"name": "a", "description": strings.Repeat("x", minCacheBytes)},
		{"name": "b", "description": "small"},
	}

	marked := cacheableTools(tools, "   ", false)
	if _, ok := marked[len(marked)-1]["cache_control"]; !ok {
		t.Fatal("the last tool carries no cache_control, so a tool-only prefix is never cached")
	}
	if _, ok := tools[1]["cache_control"]; ok {
		t.Error("the caller's tool map was mutated")
	}
	if _, ok := marked[0]["cache_control"]; ok {
		t.Error("only the last tool should carry the breakpoint")
	}

	// With a system prompt present, the system breakpoint covers the tools and
	// they must not be marked.
	unmarked := cacheableTools(tools, "You are an assistant.", false)
	if _, ok := unmarked[len(unmarked)-1]["cache_control"]; ok {
		t.Error("tools were marked even though the system breakpoint covers them")
	}
}

// WithoutCache turns the whole mechanism off: system stays a plain string and
// tools are never marked, whatever their size.
func TestNoCacheDisablesEveryBreakpoint(t *testing.T) {
	big := []map[string]any{{"name": "a", "description": strings.Repeat("x", minCacheBytes)}}

	if _, plain := cacheableSystem("You are an assistant.", big, true).(string); !plain {
		t.Error("system was wrapped for caching despite NoCache")
	}
	marked := cacheableTools(big, "", true)
	if _, ok := marked[len(marked)-1]["cache_control"]; ok {
		t.Error("tools were marked despite NoCache")
	}
}
