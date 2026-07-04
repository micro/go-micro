package agent

import (
	"testing"

	"go-micro.dev/v6/ai"
)

func TestParseTextToolCallsMiniMaxTaggedMarkup(t *testing.T) {
	tools := []ai.Tool{{Name: "task_TaskService_Add"}}
	reply := `<tool_calls>
<tool_call>{"name":"task_TaskService_Add","arguments":{"title":"Design"}}</tool_call>
<tool_call>{"name":"task_TaskService_Add","arguments":{"title":"Build"}}</tool_call>
<tool_call>{"name":"task_TaskService_Add","arguments":{"title":"Ship"}}</tool_call>
</tool_calls>`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 3 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 3: %+v", len(calls), calls)
	}
	for i, want := range []string{"Design", "Build", "Ship"} {
		if calls[i].Name != "task_TaskService_Add" {
			t.Fatalf("call %d name = %q, want task_TaskService_Add", i, calls[i].Name)
		}
		if got := calls[i].Input["title"]; got != want {
			t.Fatalf("call %d title = %v, want %q", i, got, want)
		}
	}
}

func TestParseTextToolCallsFunctionTaggedMarkup(t *testing.T) {
	tools := []ai.Tool{{Name: "task_TaskService_Add"}}
	reply := `<function=task_TaskService_Add>{"title":"Design"}</function>`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 1 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 1: %+v", len(calls), calls)
	}
	if got := calls[0].Input["title"]; got != "Design" {
		t.Fatalf("title = %v, want Design", got)
	}
}
