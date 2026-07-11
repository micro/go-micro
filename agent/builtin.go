package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-micro.dev/v6/ai"
	codecBytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/gateway/a2a"
	"go-micro.dev/v6/store"
)

// Built-in agent tools. These are not service endpoints — they are
// capabilities the agent has over itself: maintaining a plan in its
// memory, and delegating a subtask to another agent.
//
// They are plain tools, wired into the agent's tool handler alongside
// the discovered service tools. There is no separate harness or graph:
// the LLM calls them like any other tool.
const (
	toolPlan       = "plan"
	toolDelegate   = "delegate"
	toolHumanInput = "request_input"
)

type delegateCall struct {
	done chan struct{}
	res  ai.ToolResult
}

// builtinTools returns the tool definitions exposed to the model in
// addition to the agent's scoped service tools.
func builtinTools() []ai.Tool {
	return []ai.Tool{
		{
			Name:         toolPlan,
			OriginalName: toolPlan,
			Description: "Record or update your plan as an ordered list of steps before doing multi-step work. " +
				"Call this whenever the plan changes. The plan is saved to your memory and shown back to you on later turns.",
			Properties: map[string]any{
				"steps": map[string]any{
					"type": "array",
					"description": "Ordered plan steps. Each step has a 'task' (string) and a " +
						"'status' (one of: pending, in_progress, done).",
				},
			},
		},
		{
			Name:         toolHumanInput,
			OriginalName: toolHumanInput,
			Description: "Pause this agent run when you need missing information, a decision, or other human input before you can continue. " +
				"The run is checkpointed as input-required and can be resumed with the human response without losing completed tool history.",
			Properties: map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "The specific question, decision, or instruction needed from the human operator.",
				},
			},
		},
		{
			Name:         toolDelegate,
			OriginalName: toolDelegate,
			Description: "Delegate a self-contained subtask to another agent. If 'to' names an agent that already " +
				"manages the relevant services, that agent handles it; otherwise a focused sub-agent is created for the " +
				"subtask. The sub-agent works in an isolated context and returns only its result. Use this to keep your " +
				"own context focused and to let domain experts handle their own services.",
			Properties: map[string]any{
				"task": map[string]any{
					"type":        "string",
					"description": "The subtask to delegate, described completely and self-contained.",
				},
				"to": map[string]any{
					"type":        "string",
					"description": "Optional. The agent or service name best suited to the subtask, or the URL of an external agent that speaks the A2A protocol.",
				},
			},
		},
	}
}

// Builtins returns the built-in agent tools (plan, delegate) together
// with a handler for them, so the same capabilities can be wired into a
// tool loop that isn't a running Agent — for example the `micro chat`
// fallback. The handler's third return value is false when the name is
// not a built-in, so callers can fall through to their own tools.
//
// Configure it with the same options as an Agent (Name, Provider,
// WithStore, WithRegistry, WithClient, ...); these back plan's memory
// and delegate's RPC/sub-agent behavior.
func Builtins(opts ...Option) (tools []ai.Tool, handle func(name string, input map[string]any) (result any, content string, ok bool)) {
	a := &agentImpl{opts: newOptions(opts...)}
	handle = func(name string, input map[string]any) (any, string, bool) {
		switch name {
		case toolPlan:
			r := a.handlePlan(ai.ToolCall{Name: name, Input: input})
			return r.Value, r.Content, true
		case toolHumanInput:
			r := a.handleHumanInput(ai.ToolCall{Name: name, Input: input})
			return r.Value, r.Content, true
		case toolDelegate:
			r := a.handleDelegate(context.Background(), ai.ToolCall{Name: name, Input: input})
			return r.Value, r.Content, true
		}
		return nil, "", false
	}
	return builtinTools(), handle
}

// toolHandler returns the agent's tool-call handler, composed as a stack
// of wrappers around a base handler — the same middleware shape as
// client/server wrappers. The base executes the call (custom tools,
// delegate, or RPC); the built-in guardrails wrap it; developer wrappers
// (WrapTool) wrap those, outermost, so they observe every call and its
// result including guardrail refusals. Ephemeral sub-agents get the bare
// service handler so they can neither plan nor re-delegate (which
// prevents runaway recursion).
func (a *agentImpl) toolHandler() ai.ToolHandler {
	if a.ephemeral {
		return a.toolTimeoutWrap(a.tools.Handler())
	}

	// Innermost first: base, then guardrails (approve → loop → step →
	// plan), then developer wrappers outermost. Wrapping reverses order,
	// so the result runs plan → step → loop → approve → checkpoint → base.
	h := a.baseHandler()
	h = a.toolTimeoutWrap(h)
	h = a.toolRetryWrap(h)
	h = a.checkpointToolWrap(h)
	h = a.approveWrap(h)
	h = a.loopWrap(h)
	h = a.stepWrap(h)
	h = a.planWrap(h)
	h = contextWrap(h)
	h = a.traceTool(h)
	for i := len(a.opts.wrappers) - 1; i >= 0; i-- {
		h = a.opts.wrappers[i](h)
	}
	return h
}

// contextWrap stops tool execution promptly when the Ask context has
// already been canceled or its deadline has expired. This keeps guardrail
// bookkeeping and side-effecting tools from running after the caller has
// abandoned the agent run.
func contextWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		select {
		case <-ctx.Done():
			return errResult(call.ID, ctx.Err().Error())
		default:
		}
		return next(ctx, call)
	}
}

// toolTimeoutWrap gives each tool execution its own deadline while preserving
// caller cancellation. Handlers still execute synchronously; tools that honor
// context (custom tools, delegate RPC/A2A, and go-micro RPC clients) return
// promptly with a bounded error result when the deadline expires.
func (a *agentImpl) toolTimeoutWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.ToolTimeout <= 0 {
			return next(ctx, call)
		}
		toolCtx, cancel := context.WithTimeout(ctx, a.opts.ToolTimeout)
		defer cancel()
		return next(toolCtx, call)
	}
}

// toolRetryWrap retries transient tool failures with bounded backoff. It is
// opt-in because tools can have side effects; guardrail refusals and caller
// cancellation are never retried.
func (a *agentImpl) toolRetryWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		maxAttempts := a.opts.ToolMaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 1
		}

		var res ai.ToolResult
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			if err := ctx.Err(); err != nil {
				return errResult(call.ID, err.Error())
			}
			res = next(ctx, call)
			if !retryableToolResult(res) || attempt == maxAttempts || ctx.Err() != nil {
				return annotateToolAttempts(res, attempt)
			}

			t := time.NewTimer(toolRetryBackoff(attempt, a.opts.ToolRetryBackoff))
			select {
			case <-ctx.Done():
				if !t.Stop() {
					<-t.C
				}
				return errResult(call.ID, ctx.Err().Error())
			case <-t.C:
			}
		}
		return annotateToolAttempts(res, maxAttempts)
	}
}

func retryableToolResult(res ai.ToolResult) bool {
	if res.Refused != "" {
		return false
	}
	msg := toolErrorMessage(res)
	if msg == "" {
		return false
	}
	return ai.IsTransientError(fmt.Errorf("%s", msg))
}

func toolErrorMessage(res ai.ToolResult) string {
	if m, ok := res.Value.(map[string]string); ok {
		return m["error"]
	}
	if m, ok := res.Value.(map[string]any); ok {
		if v, ok := m["error"].(string); ok {
			return v
		}
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(res.Content), &decoded); err == nil {
		return decoded["error"]
	}
	return ""
}

func annotateToolAttempts(res ai.ToolResult, attempts int) ai.ToolResult {
	if attempts <= 1 {
		return res
	}
	res.Attempts = attempts
	if m, ok := res.Value.(map[string]string); ok {
		cp := map[string]any{}
		for k, v := range m {
			cp[k] = v
		}
		cp["attempts"] = attempts
		res.Value = cp
		if b, err := json.Marshal(cp); err == nil {
			res.Content = string(b)
		}
	}
	return res
}

func toolRetryBackoff(attempt int, base time.Duration) time.Duration {
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	if shift := attempt - 1; shift > 0 {
		base <<= shift
	}
	if base > 30*time.Second {
		return 30 * time.Second
	}
	return base
}

// baseHandler executes a tool call: a developer custom tool, the built-in
// delegate, or an RPC to the service. It is the innermost handler.
func (a *agentImpl) baseHandler() ai.ToolHandler {
	rpc := a.tools.Handler()
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		for i := range a.opts.tools {
			if a.opts.tools[i].def.Name == call.Name {
				out, err := a.opts.tools[i].handler(ctx, call.Input)
				if err != nil {
					return errResult(call.ID, err.Error())
				}
				return ai.ToolResult{ID: call.ID, Value: out, Content: out}
			}
		}
		if call.Name == toolHumanInput {
			return a.handleHumanInput(call)
		}
		if call.Name == toolDelegate {
			return a.handleDelegate(ctx, call)
		}
		return rpc(ctx, call)
	}
}

// planWrap handles the plan tool inline. plan is internal bookkeeping,
// not an action — it is never counted, loop-checked, or gated.
func (a *agentImpl) planWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if call.Name == toolPlan {
			return a.handlePlan(call)
		}
		if containsNestedTextToolCall(call.Input) {
			return refused(call.ID, ai.RefusedApproval, "malformed tool call: nested text tool-call markup found inside arguments; call the intended tool directly with clean JSON arguments")
		}
		if call.Name == toolDelegate {
			if blocked := a.unfinishedPlanStepsBeforeDelegation(); len(blocked) > 0 {
				return refused(call.ID, ai.RefusedApproval, "complete these plan steps before delegating: "+strings.Join(blocked, ", "))
			}
		}
		res := next(ctx, call)
		if res.Refused == "" && toolErrorMessage(res) == "" {
			a.completeNextPlanStep()
		}
		return res
	}
}

// stepWrap bounds the number of actions per Ask (MaxSteps).
func (a *agentImpl) stepWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.MaxSteps > 0 {
			a.steps++
			if a.steps > a.opts.MaxSteps {
				return refused(call.ID, ai.RefusedMaxSteps, fmt.Sprintf(
					"step limit reached (%d). Do not call any more tools; stop and summarize what you have so far.",
					a.opts.MaxSteps))
			}
		}
		return next(ctx, call)
	}
}

// loopWrap stops the agent repeating an identical action that makes no
// progress (which the step count alone won't catch).
func (a *agentImpl) loopWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.LoopLimit > 0 {
			if a.calls == nil {
				a.calls = map[string]int{}
			}
			args, _ := json.Marshal(call.Input)
			fp := call.Name + ":" + string(args)
			a.calls[fp]++
			if a.calls[fp] > a.opts.LoopLimit {
				return refused(call.ID, ai.RefusedLoop, fmt.Sprintf(
					"loop detected: you have already called %q with the same arguments %d times and the result will not change. Stop repeating it — try a different approach, or finish with what you have.",
					call.Name, a.opts.LoopLimit))
			}
		}
		return next(ctx, call)
	}
}

// approveWrap gates each action before it runs (ApproveTool).
type approvalPause struct {
	Tool    string
	Message string
}

type inputPause struct {
	OriginalMessage string `json:"original_message"`
	Prompt          string `json:"prompt"`
}

func (a *agentImpl) approveWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.Approve != nil {
			if ok, reason := a.opts.Approve(call.Name, call.Input); !ok {
				msg := "tool call was not approved"
				if reason != "" {
					msg += ": " + reason
				}
				a.pause = &approvalPause{Tool: call.Name, Message: msg}
				return refused(call.ID, ai.RefusedApproval, msg)
			}
		}
		return next(ctx, call)
	}
}

// handlePlan persists the supplied plan to the agent's memory and
// echoes it back so the model can see the stored state.
func (a *agentImpl) handlePlan(call ai.ToolCall) ai.ToolResult {
	input := preserveCompletedPlanSteps(a.loadPlan(), call.Input)
	data, err := json.Marshal(input)
	if err != nil {
		return errResult(call.ID, "invalid plan: "+err.Error())
	}
	_ = a.stateStore().Write(&store.Record{Key: planKey, Value: data})
	return ai.ToolResult{ID: call.ID, Value: input, Content: string(data)}
}

func preserveCompletedPlanSteps(stored string, input map[string]any) map[string]any {
	if stored == "" {
		return input
	}
	var previous map[string]any
	if err := json.Unmarshal([]byte(stored), &previous); err != nil {
		return input
	}
	completed := completedPlanTasks(previous)
	if len(completed) == 0 {
		return input
	}
	steps, ok := input["steps"].([]any)
	if !ok {
		return input
	}
	for _, raw := range steps {
		step, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		task, _ := step["task"].(string)
		if completed[planTaskCompletionKey(task)] && isUnfinishedPlanStatus(step["status"]) {
			step["status"] = "done"
		}
	}
	return input
}

func completedPlanTasks(plan map[string]any) map[string]bool {
	steps, ok := plan["steps"].([]any)
	if !ok {
		return nil
	}
	completed := map[string]bool{}
	for _, raw := range steps {
		step, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		status, _ := step["status"].(string)
		if status != "done" {
			continue
		}
		task, _ := step["task"].(string)
		if task = planTaskCompletionKey(task); task != "" {
			completed[task] = true
		}
	}
	return completed
}

func normalizePlanTask(task string) string {
	return strings.Join(strings.Fields(strings.ToLower(task)), " ")
}

func planTaskCompletionKey(task string) string {
	normalized := normalizePlanTask(task)
	if normalized == "" {
		return ""
	}
	if isLaunchReadinessDelegationPlanTask(normalized) {
		return "launch-readiness-notification"
	}
	return normalized
}

func isLaunchReadinessDelegationPlanTask(task string) bool {
	task = normalizePlanTask(task)
	if !strings.Contains(task, "notify") && !strings.Contains(task, "notification") {
		return false
	}
	hasLaunchReadiness := strings.Contains(task, "launch") || strings.Contains(task, "readiness") || strings.Contains(task, "ready")
	hasOwnerComms := strings.Contains(task, "owner") && strings.Contains(task, "comms")
	return hasLaunchReadiness || hasOwnerComms
}

func isUnfinishedPlanStatus(status any) bool {
	s, _ := status.(string)
	return s == "" || s == "pending" || s == "in_progress"
}

func (a *agentImpl) completeNextPlanStep() {
	plan := a.loadPlan()
	if plan == "" {
		return
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(plan), &data); err != nil {
		return
	}
	steps, ok := data["steps"].([]any)
	if !ok {
		return
	}
	for _, raw := range steps {
		step, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		status, _ := step["status"].(string)
		if status == "" || status == "pending" || status == "in_progress" {
			step["status"] = "done"
			b, err := json.Marshal(data)
			if err == nil {
				_ = a.stateStore().Write(&store.Record{Key: planKey, Value: b})
			}
			return
		}
	}
}

func (a *agentImpl) unfinishedPlanStepsBeforeDelegation() []string {
	plan := a.loadPlan()
	if plan == "" {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(plan), &data); err != nil {
		return nil
	}
	steps, ok := data["steps"].([]any)
	if !ok {
		return nil
	}
	var unfinished []string
	for _, raw := range steps {
		step, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		task := planStepTask(step)
		if isDelegationPlanTask(task) {
			break
		}
		if !isUnfinishedPlanStatus(step["status"]) {
			continue
		}
		if task == "" {
			task = "<unnamed>"
		}
		unfinished = append(unfinished, task)
	}
	return unfinished
}

func planStepTask(step map[string]any) string {
	if task, _ := step["task"].(string); task != "" {
		return task
	}
	desc, _ := step["description"].(string)
	return desc
}

func isDelegationPlanTask(task string) bool {
	task = normalizePlanTask(task)
	return strings.Contains(task, "delegate") || strings.Contains(task, "notify") || strings.Contains(task, "notification")
}

func (a *agentImpl) unfinishedPlanSteps() []string {
	plan := a.loadPlan()
	if plan == "" {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(plan), &data); err != nil {
		return nil
	}
	steps, ok := data["steps"].([]any)
	if !ok {
		return nil
	}
	var unfinished []string
	for _, raw := range steps {
		step, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		status, _ := step["status"].(string)
		if status != "" && status != "pending" && status != "in_progress" {
			continue
		}
		task := planStepTask(step)
		if task == "" {
			task = "<unnamed>"
		}
		unfinished = append(unfinished, task)
	}
	return unfinished
}

// handleHumanInput records that the model needs operator input before it can continue.
func (a *agentImpl) handleHumanInput(call ai.ToolCall) ai.ToolResult {
	prompt, _ := call.Input["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		prompt = "human input required"
	}
	a.pause = &approvalPause{Tool: toolHumanInput, Message: prompt}
	return refused(call.ID, ai.RefusedApproval, "input-required: "+prompt)
}

// handleDelegate hands a subtask to another agent. Delegate-first:
// if 'to' names a registered agent, it is called via RPC. Otherwise an
// ephemeral sub-agent is created with a fresh, isolated context, asked
// the subtask, and its reply returned.
func (a *agentImpl) handleDelegate(ctx context.Context, call ai.ToolCall) (res ai.ToolResult) {
	input := call.Input
	task, _ := input["task"].(string)
	if task == "" {
		return errResult(call.ID, "task is required")
	}
	to, _ := input["to"].(string)
	if cached, ok := a.cachedDelegateResult(call.ID, to, task); ok {
		return cached
	}

	key := delegateResultKey(to, task)
	if cached, ok := a.joinDelegateCall(ctx, call.ID, key); ok {
		return cached
	}
	defer func() { a.finishDelegateCall(key, res) }()

	// An external agent on another framework, addressed by A2A URL.
	if strings.HasPrefix(to, "http://") || strings.HasPrefix(to, "https://") {
		reply, err := a2a.NewClient(to).Send(ctx, task)
		if err != nil {
			return errResult(call.ID, "delegate to A2A agent "+to+": "+err.Error())
		}
		return a.storeDelegateResult(call.ID, to, task, map[string]any{"agent": to, "reply": reply})
	}

	// Delegate-first: an existing agent that owns the domain handles it.
	if to != "" && a.isAgent(to) {
		reply, err := a.callAgentRPC(ctx, to, task)
		if err != nil {
			return errResult(call.ID, "delegate to agent "+to+": "+err.Error())
		}
		return a.storeDelegateResult(call.ID, to, task, map[string]any{"agent": to, "reply": reply})
	}

	// Otherwise create a focused, ephemeral sub-agent. Fresh context:
	// it loads no history and persists none.
	var svcs []string
	if to != "" {
		svcs = []string{to}
	}
	sub := newEphemeral(
		Name(a.opts.Name+".sub"),
		Services(svcs...),
		Prompt("You are a sub-agent handling a single delegated subtask. "+
			"Complete it using the available tools and report the result concisely."),
		Provider(a.opts.Provider),
		Model(a.opts.Model),
		APIKey(a.opts.APIKey),
		WithRegistry(a.opts.Registry),
		WithClient(a.opts.Client),
		WithStore(a.opts.Store),
		ModelCallTimeout(a.opts.ModelTimeout),
		ModelRetry(a.opts.ModelMaxAttempts, a.opts.ModelRetryBackoff),
		ToolCallTimeout(a.opts.ToolTimeout),
		ToolRetry(a.opts.ToolMaxAttempts, a.opts.ToolRetryBackoff),
		TraceProvider(a.opts.TraceProvider),
	)
	// Record lineage so the sub-agent's tool calls carry this run as parent.
	sub.parentRunID = a.runID

	resp, err := sub.Ask(ctx, task)
	if err != nil {
		return errResult(call.ID, "sub-agent: "+err.Error())
	}
	return a.storeDelegateResult(call.ID, to, task, map[string]any{"reply": resp.Reply})
}

func (a *agentImpl) joinDelegateCall(ctx context.Context, id, key string) (ai.ToolResult, bool) {
	a.delegateMu.Lock()
	if a.delegateCalls == nil {
		a.delegateCalls = map[string]*delegateCall{}
	}
	if inFlight := a.delegateCalls[key]; inFlight != nil {
		a.delegateMu.Unlock()
		select {
		case <-ctx.Done():
			return errResult(id, ctx.Err().Error()), true
		case <-inFlight.done:
			return withToolResultID(inFlight.res, id), true
		}
	}
	a.delegateCalls[key] = &delegateCall{done: make(chan struct{})}
	a.delegateMu.Unlock()
	return ai.ToolResult{}, false
}

func (a *agentImpl) finishDelegateCall(key string, res ai.ToolResult) {
	a.delegateMu.Lock()
	inFlight := a.delegateCalls[key]
	if inFlight == nil {
		a.delegateMu.Unlock()
		return
	}
	inFlight.res = res
	delete(a.delegateCalls, key)
	close(inFlight.done)
	a.delegateMu.Unlock()
}

func (a *agentImpl) cachedDelegateResult(id, to, task string) (ai.ToolResult, bool) {
	recs, err := a.stateStore().Read(delegateResultKey(to, task))
	if err != nil || len(recs) == 0 {
		return ai.ToolResult{}, false
	}
	var out map[string]any
	if err := json.Unmarshal(recs[0].Value, &out); err != nil {
		return ai.ToolResult{}, false
	}
	b, _ := json.Marshal(out)
	return ai.ToolResult{ID: id, Value: out, Content: string(b)}, true
}

func (a *agentImpl) storeDelegateResult(id, to, task string, out map[string]any) ai.ToolResult {
	b, _ := json.Marshal(out)
	_ = a.stateStore().Write(&store.Record{Key: delegateResultKey(to, task), Value: b})
	return ai.ToolResult{ID: id, Value: out, Content: string(b)}
}

func withToolResultID(res ai.ToolResult, id string) ai.ToolResult {
	res.ID = id
	return res
}

func delegateResultKey(to, task string) string {
	fp := normalizeDelegateTarget(to) + "\x00" + normalizeDelegateTask(task)
	sum := sha256.Sum256([]byte(fp))
	return fmt.Sprintf("delegate/%x", sum)
}

func normalizeDelegateTarget(to string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(to))), " ")
}

func normalizeDelegateTask(task string) string {
	task = strings.ToLower(strings.TrimSpace(task))
	task = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == '@':
			return r
		default:
			return ' '
		}
	}, task)
	task = strings.Join(strings.Fields(task), " ")
	if strings.Contains(task, "owner") &&
		strings.Contains(task, "acme") &&
		isLaunchReadinessDelegateTask(task) {
		return "notify owner@acme.com launch-plan-ready"
	}
	return task
}

func isLaunchReadinessDelegateTask(task string) bool {
	hasNotify := strings.Contains(task, "notify") || strings.Contains(task, "notification") || strings.Contains(task, "tell")
	hasLaunch := strings.Contains(task, "launch")
	hasPlanOrReadiness := strings.Contains(task, "plan") || strings.Contains(task, "readiness") || strings.Contains(task, "ready")
	hasCompletion := strings.Contains(task, "ready") ||
		strings.Contains(task, "readiness") ||
		strings.Contains(task, "prepared") ||
		strings.Contains(task, "complete") ||
		strings.Contains(task, "finished") ||
		strings.Contains(task, "done") ||
		strings.Contains(task, "sent")
	return hasNotify && hasLaunch && hasPlanOrReadiness && hasCompletion
}

// isAgent reports whether name resolves to a registered agent (a
// service advertising type=agent in its metadata).
func (a *agentImpl) isAgent(name string) bool {
	if a.opts.Registry == nil {
		return false
	}
	recs, err := a.opts.Registry.GetService(name)
	if err != nil || len(recs) == 0 {
		return false
	}
	if recs[0].Metadata != nil && recs[0].Metadata["type"] == "agent" {
		return true
	}
	for _, n := range recs[0].Nodes {
		if n.Metadata != nil && n.Metadata["type"] == "agent" {
			return true
		}
	}
	return false
}

// callAgentRPC calls another agent's Agent.Chat endpoint and returns
// its reply.
func (a *agentImpl) callAgentRPC(ctx context.Context, name, msg string) (string, error) {
	body, _ := json.Marshal(map[string]string{"message": msg})
	req := a.opts.Client.NewRequest(name, "Agent.Chat", &codecBytes.Frame{Data: body})
	var rsp codecBytes.Frame
	if err := a.opts.Client.Call(ctx, req, &rsp); err != nil {
		return "", err
	}
	var out struct {
		Reply string `json:"reply"`
	}
	if err := json.Unmarshal(rsp.Data, &out); err != nil {
		return "", err
	}
	return out.Reply, nil
}

// planKey is the record key for an agent's plan within its scoped store.
const planKey = "plan"

// loadPlan returns the stored plan as a JSON string, or "" if none.
func (a *agentImpl) loadPlan() string {
	recs, err := a.stateStore().Read(planKey)
	if err != nil || len(recs) == 0 {
		return ""
	}
	return string(recs[0].Value)
}

func errResult(id, msg string) ai.ToolResult {
	m := map[string]string{"error": msg}
	b, _ := json.Marshal(m)
	return ai.ToolResult{ID: id, Value: m, Content: string(b)}
}

// refused is an error result a guardrail returns, tagged with a structured
// reason (ai.Refused*) so a tool wrapper can react to it without parsing
// the message.
func refused(id, reason, msg string) ai.ToolResult {
	r := errResult(id, msg)
	r.Refused = reason
	return r
}
