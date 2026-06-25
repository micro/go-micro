package flow

import "time"

// Options configures a Flow.
type Options struct {
	// TriggerTopic is the broker topic that triggers this flow.
	TriggerTopic string
	// Prompt is a Go template string. {{.Data}} is the event payload.
	Prompt string
	// SystemPrompt is the system instruction for the LLM.
	SystemPrompt string
	// Provider is the AI provider name (e.g. "anthropic", "openai").
	Provider string
	// APIKey for the AI provider.
	APIKey string
	// Model overrides the provider's default model.
	Model string
	// BaseURL overrides the provider's default base URL.
	BaseURL string
	// HistoryLimit is the max messages per flow execution.
	HistoryLimit int
	// OnResult is called after each execution with the result.
	OnResult func(Result)
	// Agent, if set, names a registered agent the flow hands each event
	// to (over RPC). The flow triggers; the agent reasons. When empty,
	// the flow runs a single augmented-LLM step itself.
	Agent string

	// Steps, if set, makes the flow run an ordered list of steps per
	// event instead of a single LLM step — the deterministic-workflow
	// path. Checkpointed between steps when a Checkpoint is set.
	Steps []Step
	// Retry is the flow-level retry count applied to each step (0 = no
	// retry). A Step's own Retry field overrides this.
	Retry int
	// RetryBackoff is the delay between failed step attempts. Zero means
	// retry immediately; cancellation/deadline stops the wait early.
	RetryBackoff time.Duration
	// Checkpoint is the durability backend for stepped runs. Nil with
	// steps present means a store-backed default; set it to swap backends.
	Checkpoint Checkpoint
	// DeleteOnSuccess removes a run's checkpoint when it completes
	// successfully. Failed runs are always retained. Default: retain all.
	DeleteOnSuccess bool
}

// Option applies a configuration to Options.
type Option func(*Options)

// Trigger sets the broker topic that triggers this flow.
func Trigger(topic string) Option {
	return func(o *Options) { o.TriggerTopic = topic }
}

// Prompt sets the prompt template. Use {{.Data}} for the event payload.
func Prompt(p string) Option {
	return func(o *Options) { o.Prompt = p }
}

// SystemPrompt sets the system instruction for the LLM.
func SystemPrompt(p string) Option {
	return func(o *Options) { o.SystemPrompt = p }
}

// Provider sets the AI provider name.
func Provider(name string) Option {
	return func(o *Options) { o.Provider = name }
}

// APIKey sets the API key for the AI provider.
func APIKey(key string) Option {
	return func(o *Options) { o.APIKey = key }
}

// Model sets the model name.
func Model(name string) Option {
	return func(o *Options) { o.Model = name }
}

// BaseURL sets the provider base URL.
func BaseURL(url string) Option {
	return func(o *Options) { o.BaseURL = url }
}

// HistoryLimit sets the max messages per execution.
func HistoryLimit(n int) Option {
	return func(o *Options) { o.HistoryLimit = n }
}

// OnResult sets a callback for each execution result.
func OnResult(fn func(Result)) Option {
	return func(o *Options) { o.OnResult = fn }
}

// Agent makes the flow hand each event to a named registered agent over
// RPC instead of running its own LLM step. The flow triggers; the agent
// reasons (with its plan, delegate, memory, and guardrails).
func Agent(name string) Option {
	return func(o *Options) { o.Agent = name }
}

// Steps sets the ordered steps of the flow. A flow with steps runs them
// in order per event, checkpointing between each, instead of the
// single-step prompt/agent behavior. Step names must be unique.
func Steps(steps ...Step) Option {
	return func(o *Options) { o.Steps = steps }
}

// Retry sets the flow-level retry count applied to each step (0 = no
// retry). A Step's own Retry field overrides this.
func Retry(n int) Option {
	return func(o *Options) { o.Retry = n }
}

// RetryBackoff sets the delay between failed step attempts. A zero
// duration preserves immediate retries. If the run context is canceled
// while waiting, the context error is returned instead of retrying.
func RetryBackoff(d time.Duration) Option {
	return func(o *Options) { o.RetryBackoff = d }
}

// WithCheckpoint sets the durability backend. With a checkpoint, a run is
// persisted before and after each step and can be resumed after a crash.
// Stepped flows default to a store-backed checkpoint; use this to swap it.
func WithCheckpoint(c Checkpoint) Option {
	return func(o *Options) { o.Checkpoint = c }
}

// DeleteOnSuccess removes a run's checkpoint when it completes
// successfully. Failed runs are always retained. Default: retain all.
func DeleteOnSuccess() Option {
	return func(o *Options) { o.DeleteOnSuccess = true }
}
