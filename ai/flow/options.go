package flow

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
