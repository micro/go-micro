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

func TestParseTextToolCallsCreateAliasForAddTool(t *testing.T) {
	tools := []ai.Tool{{Name: "task_TaskService_Add", OriginalName: "task.TaskService.Add"}}
	reply := `<tool_call>{"name":"task_TaskService_Create","arguments":{"title":"Design"}}</tool_call>`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 1 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 1: %+v", len(calls), calls)
	}
	if calls[0].Name != "task_TaskService_Add" {
		t.Fatalf("call name = %q, want canonical task_TaskService_Add", calls[0].Name)
	}
	if got := calls[0].Input["title"]; got != "Design" {
		t.Fatalf("title = %v, want Design", got)
	}
}

func TestParseTextToolCallsOpenAICompatibleFunctionArgumentsString(t *testing.T) {
	tools := []ai.Tool{{Name: "delegate"}}
	reply := `<tool_call>{"id":"call-2","type":"function","function":{"name":"delegate","arguments":"{\"task\":\"summarize the conformance marker\",\"to\":\"blocked-reviewer\"}"}}</tool_call>`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 1 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 1: %+v", len(calls), calls)
	}
	if calls[0].Name != "delegate" {
		t.Fatalf("call name = %q, want delegate", calls[0].Name)
	}
	if got := calls[0].Input["task"]; got != "summarize the conformance marker" {
		t.Fatalf("task = %v, want summarize the conformance marker", got)
	}
	if got := calls[0].Input["to"]; got != "blocked-reviewer" {
		t.Fatalf("to = %v, want blocked-reviewer", got)
	}
}

func TestParseTextToolCallsTaggedMarkupWithSpacedNameAttribute(t *testing.T) {
	tools := []ai.Tool{{Name: "delegate"}}
	reply := `<tool_call name = "delegate">{"task":"summarize the conformance marker","to":"blocked-reviewer"}</tool_call>`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 1 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 1: %+v", len(calls), calls)
	}
	if calls[0].Name != "delegate" {
		t.Fatalf("call name = %q, want delegate", calls[0].Name)
	}
	if got := calls[0].Input["to"]; got != "blocked-reviewer" {
		t.Fatalf("to = %v, want blocked-reviewer", got)
	}
}

func TestParseTextToolCallsHTMLEscapedTaggedMarkup(t *testing.T) {
	tools := []ai.Tool{{Name: "delegate"}}
	reply := `&lt;tool_call name=&quot;delegate&quot;&gt;{"task":"summarize the conformance marker","to":"blocked-reviewer"}&lt;/tool_call&gt;`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 1 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 1: %+v", len(calls), calls)
	}
	if calls[0].Name != "delegate" {
		t.Fatalf("call name = %q, want delegate", calls[0].Name)
	}
	if got := calls[0].Input["task"]; got != "summarize the conformance marker" {
		t.Fatalf("task = %v, want summarize the conformance marker", got)
	}
}

func TestParseTextToolCallsFunctionCallSyntax(t *testing.T) {
	tools := []ai.Tool{{Name: "delegate"}}
	reply := `I will now call delegate({"task":"summarize the conformance marker","to":"blocked-reviewer"}) before answering.`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 1 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 1: %+v", len(calls), calls)
	}
	if calls[0].Name != "delegate" {
		t.Fatalf("call name = %q, want delegate", calls[0].Name)
	}
	if got := calls[0].Input["task"]; got != "summarize the conformance marker" {
		t.Fatalf("task = %v, want summarize the conformance marker", got)
	}
	if got := calls[0].Input["to"]; got != "blocked-reviewer" {
		t.Fatalf("to = %v, want blocked-reviewer", got)
	}
}

func TestParseTextToolCallsFunctionCallSyntaxHandlesNestedJSON(t *testing.T) {
	tools := []ai.Tool{{Name: "delegate"}}
	reply := `delegate({
		"task":"summarize the {escaped} marker",
		"meta":{"note":"paren ) and brace } in string"},
		"to":"blocked-reviewer"
	})`

	calls := parseTextToolCalls(reply, tools)
	if len(calls) != 1 {
		t.Fatalf("parseTextToolCalls returned %d calls, want 1: %+v", len(calls), calls)
	}
	if got := calls[0].Input["task"]; got != "summarize the {escaped} marker" {
		t.Fatalf("task = %v, want nested JSON-safe task", got)
	}
	if got := calls[0].Input["to"]; got != "blocked-reviewer" {
		t.Fatalf("to = %v, want blocked-reviewer", got)
	}
}
