package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v5/model"
	"go-micro.dev/v5/registry"
)

// TestNew verifies that New returns an Agent with default options applied.
func TestNew(t *testing.T) {
	a := New()
	require.NotNil(t, a)
	assert.Equal(t, "agent", a.String())

	opts := a.Options()
	assert.NotEmpty(t, opts.Directive)
	assert.Equal(t, 30*time.Second, opts.Interval)
	assert.NotNil(t, opts.Context)
}

// TestNewWithOptions verifies functional options are applied correctly.
func TestNewWithOptions(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	a := New(
		WithName("test-agent"),
		WithDirective("manage my service"),
		WithServices("svc-a", "svc-b"),
		WithRegistry(reg),
		WithInterval(5*time.Second),
	)

	require.NotNil(t, a)
	assert.Equal(t, "test-agent", a.String())

	opts := a.Options()
	assert.Equal(t, "manage my service", opts.Directive)
	assert.Equal(t, []string{"svc-a", "svc-b"}, opts.Services)
	assert.Equal(t, 5*time.Second, opts.Interval)
}

// TestInit verifies Init applies additional options after creation.
func TestInit(t *testing.T) {
	a := New(WithName("orig"))
	err := a.Init(WithName("updated"))
	require.NoError(t, err)
	assert.Equal(t, "updated", a.String())
}

// TestRunStop verifies Run starts and Stop terminates the agent cleanly.
func TestRunStop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	a := New(
		WithName("lifecycle-agent"),
		WithContext(ctx),
		WithInterval(10*time.Second), // long interval – evaluation won't run
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Run()
	}()

	// Give the goroutine a moment to start.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, a.Stop())
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("agent did not stop in time")
	}
}

// TestServiceStatus verifies serviceStatus with an in-memory registry.
func TestServiceStatus(t *testing.T) {
	reg := registry.NewMemoryRegistry()

	// Register a fake service.
	err := reg.Register(&registry.Service{
		Name:    "greeter",
		Version: "1.0.0",
		Nodes: []*registry.Node{
			{Id: "greeter-1", Address: "127.0.0.1:8080"},
		},
	})
	require.NoError(t, err)

	a := &agent{
		opts: newOptions(
			WithRegistry(reg),
			WithServices("greeter", "missing-svc"),
		),
		stop: make(chan struct{}),
	}

	status, err := a.serviceStatus()
	require.NoError(t, err)
	assert.Contains(t, status, `"greeter"`)
	assert.Contains(t, status, `"running":true`)
	assert.Contains(t, status, `"missing-svc"`)
	assert.Contains(t, status, `"running":false`)
}

// TestBuildTools verifies the built-in tool definitions are well-formed.
func TestBuildTools(t *testing.T) {
	a := &agent{
		opts: newOptions(),
		stop: make(chan struct{}),
	}
	tools := a.buildTools()
	assert.Len(t, tools, 3)

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.Properties)
	}
	assert.True(t, names["list_services"])
	assert.True(t, names["get_service_status"])
	assert.True(t, names["call_service"])
}

// TestExecuteToolListServices verifies list_services returns service state.
func TestExecuteToolListServices(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	err := reg.Register(&registry.Service{
		Name:    "hello",
		Version: "v1",
		Nodes:   []*registry.Node{{Id: "hello-1", Address: "127.0.0.1:9090"}},
	})
	require.NoError(t, err)

	a := &agent{
		opts: newOptions(
			WithRegistry(reg),
			WithServices("hello"),
		),
		stop: make(chan struct{}),
	}

	_, content := a.executeTool("list_services", nil)
	assert.Contains(t, content, `"hello"`)
}

// TestExecuteToolGetServiceStatus verifies get_service_status returns details.
func TestExecuteToolGetServiceStatus(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	err := reg.Register(&registry.Service{
		Name:    "store",
		Version: "v2",
		Nodes:   []*registry.Node{{Id: "store-1", Address: "127.0.0.1:7070"}},
	})
	require.NoError(t, err)

	a := &agent{
		opts: newOptions(WithRegistry(reg)),
		stop: make(chan struct{}),
	}

	_, content := a.executeTool("get_service_status", map[string]any{"name": "store"})
	assert.Contains(t, content, `"running":true`)

	_, missing := a.executeTool("get_service_status", map[string]any{"name": "unknown"})
	assert.Contains(t, missing, `"running":`)

	_, noName := a.executeTool("get_service_status", map[string]any{})
	assert.Contains(t, noName, "error")
}

// TestExecuteToolUnknownWithHandler verifies custom tool handlers are called.
func TestExecuteToolUnknownWithHandler(t *testing.T) {
	called := false
	a := &agent{
		opts: newOptions(WithToolHandler(func(name string, input map[string]any) (any, string) {
			called = true
			return nil, `{"custom": true}`
		})),
		stop: make(chan struct{}),
	}

	_, content := a.executeTool("custom_tool", map[string]any{})
	assert.True(t, called)
	assert.Contains(t, content, "custom")
}

// TestExecuteToolUnknownNoHandler verifies unknown tools return an error when no handler is set.
func TestExecuteToolUnknownNoHandler(t *testing.T) {
	a := &agent{opts: newOptions(), stop: make(chan struct{})}
	_, content := a.executeTool("nope", nil)
	assert.Contains(t, content, "error")
}

// TestEvaluateNoModel verifies evaluate is a no-op when no model is configured.
func TestEvaluateNoModel(t *testing.T) {
	a := &agent{opts: newOptions(), stop: make(chan struct{})}
	err := a.evaluate(nil)
	assert.NoError(t, err)
}

// TestEvaluateWithMockModel verifies evaluate calls the model and handles tool calls.
func TestEvaluateWithMockModel(t *testing.T) {
	mockModel := &mockModel{
		resp: &model.Response{
			ToolCalls: []model.ToolCall{
				{Name: "list_services", Input: map[string]any{}},
			},
		},
	}

	reg := registry.NewMemoryRegistry()
	a := &agent{
		opts: newOptions(
			WithModel(mockModel),
			WithRegistry(reg),
		),
		stop: make(chan struct{}),
	}

	tools := a.buildTools()
	err := a.evaluate(tools)
	assert.NoError(t, err)
	assert.True(t, mockModel.called)
}

// TestDirectiveHelper verifies the Directive helper function.
func TestDirectiveHelper(t *testing.T) {
	a := New(WithDirective("my directive"))
	assert.Equal(t, "my directive", Directive(a))
}

// TestServicesHelper verifies the Services helper function.
func TestServicesHelper(t *testing.T) {
	a := New(WithServices("svc1", "svc2"))
	assert.Equal(t, []string{"svc1", "svc2"}, Services(a))
}

// TestWatchServicesContextCancel verifies WatchServices respects context cancellation.
func TestWatchServicesContextCancel(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- WatchServices(ctx, reg, nil, func(name string, _ *registry.Result) {})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("WatchServices did not return after context cancel")
	}
}

// TestWatchServicesNilRegistry verifies WatchServices returns error for nil registry.
func TestWatchServicesNilRegistry(t *testing.T) {
	err := WatchServices(context.Background(), nil, nil, func(string, *registry.Result) {})
	assert.Error(t, err)
}

// TestPromptNoModel verifies Prompt closes the channel immediately when no model is set.
func TestPromptNoModel(t *testing.T) {
	a := New(WithName("no-model-agent"))

	ch := a.Prompt("hello")
	select {
	case resp, ok := <-ch:
		assert.False(t, ok, "channel should be closed with no value")
		assert.Nil(t, resp)
	case <-time.After(2 * time.Second):
		t.Fatal("Prompt channel was not closed in time")
	}
}

// TestPromptNonBlocking verifies Prompt returns immediately.
func TestPromptNonBlocking(t *testing.T) {
	// slow model — blocks for up to 5 s
	slow := &slowMockModel{delay: 5 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := New(WithName("slow-agent"), WithModel(slow), WithContext(ctx))

	start := time.Now()
	ch := a.Prompt("are you there?")
	elapsed := time.Since(start)

	// Prompt must return without waiting for the model.
	assert.Less(t, elapsed, 500*time.Millisecond, "Prompt should be non-blocking")

	// Clean up: cancel context so the goroutine exits.
	cancel()
	// Drain channel.
	select {
	case <-ch:
	case <-time.After(6 * time.Second):
	}
}

// TestPromptWithModel verifies Prompt delivers the model response on the channel.
func TestPromptWithModel(t *testing.T) {
	expected := &model.Response{Reply: "all services healthy"}
	mock := &mockModel{resp: expected}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	a := New(WithName("prompt-agent"), WithModel(mock), WithContext(ctx))

	ch := a.Prompt("how are the services?")
	select {
	case resp := <-ch:
		require.NotNil(t, resp)
		assert.Equal(t, "all services healthy", resp.Reply)
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive prompt response in time")
	}

	// Channel should be closed after the single response.
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after response")
}

// TestPromptRecordsActivity verifies that Prompt records ActivityPrompt and ActivityResponse.
func TestPromptRecordsActivity(t *testing.T) {
	mock := &mockModel{resp: &model.Response{Reply: "ok"}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	a := New(WithName("activity-agent"), WithModel(mock), WithContext(ctx))

	ch := a.Prompt("status check")
	<-ch // wait for completion

	acts := a.Activity()
	require.NotEmpty(t, acts)

	types := make(map[ActivityType]int)
	for _, act := range acts {
		types[act.Type]++
		assert.False(t, act.Time.IsZero(), "activity should have a timestamp")
	}

	assert.GreaterOrEqual(t, types[ActivityPrompt], 1, "should have at least one ActivityPrompt")
	assert.GreaterOrEqual(t, types[ActivityResponse], 1, "should have at least one ActivityResponse")
}

// TestPromptRecordsToolActivity verifies tool calls made during Prompt are recorded.
func TestPromptRecordsToolActivity(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	mock := &mockModel{
		resp: &model.Response{
			Reply: "checked",
			ToolCalls: []model.ToolCall{
				{Name: "list_services", Input: map[string]any{}},
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	a := New(
		WithName("tool-activity-agent"),
		WithModel(mock),
		WithRegistry(reg),
		WithContext(ctx),
	)

	ch := a.Prompt("list services please")
	<-ch

	acts := a.Activity()
	types := make(map[ActivityType]int)
	for _, act := range acts {
		types[act.Type]++
	}
	assert.GreaterOrEqual(t, types[ActivityTool], 1, "should record at least one tool activity")
}

// TestActivityIsSnapshot verifies Activity returns an independent copy.
func TestActivityIsSnapshot(t *testing.T) {
	a := New(WithName("snapshot-agent"))

	snap1 := a.Activity()
	assert.Empty(t, snap1)

	// Directly record something.
	impl, ok := a.(*agent)
	require.True(t, ok, "New() must return *agent")
	impl.record(Activity{Type: ActivityEvaluate})

	snap2 := a.Activity()
	assert.Len(t, snap2, 1)

	// The first snapshot is unchanged.
	assert.Empty(t, snap1)
}

// TestEvaluateRecordsActivity verifies evaluate records evaluate/tool/response activities.
func TestEvaluateRecordsActivity(t *testing.T) {
	mock := &mockModel{
		resp: &model.Response{
			Reply: "evaluated",
			ToolCalls: []model.ToolCall{
				{Name: "list_services", Input: map[string]any{}},
			},
		},
	}
	reg := registry.NewMemoryRegistry()
	a := &agent{
		opts:       newOptions(WithModel(mock), WithRegistry(reg)),
		stop:       make(chan struct{}),
		activities: make([]Activity, 0, maxActivities),
	}

	tools := a.buildTools()
	err := a.evaluate(tools)
	require.NoError(t, err)

	acts := a.Activity()
	types := make(map[ActivityType]int)
	for _, act := range acts {
		types[act.Type]++
	}
	assert.GreaterOrEqual(t, types[ActivityEvaluate], 1)
	assert.GreaterOrEqual(t, types[ActivityTool], 1)
	assert.GreaterOrEqual(t, types[ActivityResponse], 1)
}

// TestPromptErrorRecorded verifies that a model error is recorded as ActivityError.
func TestPromptErrorRecorded(t *testing.T) {
	errModel := &mockModel{err: fmt.Errorf("model offline")}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	a := New(WithName("error-agent"), WithModel(errModel), WithContext(ctx))

	ch := a.Prompt("hello")
	<-ch // closed without a value on error

	acts := a.Activity()
	types := make(map[ActivityType]int)
	for _, act := range acts {
		types[act.Type]++
		if act.Type == ActivityError {
			assert.NotNil(t, act.Err)
		}
	}
	assert.GreaterOrEqual(t, types[ActivityError], 1)
}

// slowMockModel is a model.Model that blocks until its context is cancelled.
type slowMockModel struct {
	delay time.Duration
}

func (m *slowMockModel) Init(...model.Option) error { return nil }
func (m *slowMockModel) Options() model.Options     { return model.Options{} }
func (m *slowMockModel) String() string             { return "slow" }
func (m *slowMockModel) Stream(_ context.Context, _ *model.Request, _ ...model.GenerateOption) (model.Stream, error) {
	return nil, nil
}
func (m *slowMockModel) Generate(ctx context.Context, _ *model.Request, _ ...model.GenerateOption) (*model.Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(m.delay):
		return &model.Response{Reply: "done"}, nil
	}
}

// mockModel is a test double for model.Model.
type mockModel struct {
	called bool
	resp   *model.Response
	err    error
}

func (m *mockModel) Init(...model.Option) error   { return nil }
func (m *mockModel) Options() model.Options       { return model.Options{} }
func (m *mockModel) String() string               { return "mock" }
func (m *mockModel) Stream(_ context.Context, _ *model.Request, _ ...model.GenerateOption) (model.Stream, error) {
	return nil, nil
}
func (m *mockModel) Generate(_ context.Context, _ *model.Request, _ ...model.GenerateOption) (*model.Response, error) {
	m.called = true
	return m.resp, m.err
}
