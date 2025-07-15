package genai

type noopGenAI struct{}

func (n *noopGenAI) Generate(prompt string, opts ...Option) (*Result, error) {
	return &Result{Prompt: prompt, Type: "noop", Text: "noop response"}, nil
}

func (n *noopGenAI) Stream(prompt string, opts ...Option) (*Stream, error) {
	results := make(chan *Result, 1)
	results <- &Result{Prompt: prompt, Type: "noop", Text: "noop response"}
	close(results)
	return &Stream{Results: results}, nil
}

func (n *noopGenAI) String() string {
	return "noop"
}

var Default = &noopGenAI{}
