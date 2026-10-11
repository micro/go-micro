package model

import "context"

// Decider evaluates questions and returns values and probabilities.
type Decider interface {
	Init(...Option) error
	Options() Options
	Decide(context.Context, *DecisionRequest) (*DecisionResponse, error)
	String() string
}

// QuestionType identifies the kind of decision.
type QuestionType string

const (
	// Noul returns the probability that a statement is true.
	Noul QuestionType = "noul"
	// Choice selects one of the supplied options.
	Choice QuestionType = "choice"
	// Score rates input against ordered levels.
	Score QuestionType = "score"
)

// Question describes a decision. Instructions may be text, an object, or an array.
type Question struct {
	Type         QuestionType `json:"type"`
	Instructions any          `json:"instructions"`
	// Criteria is an option map for Choice, an ordered array for Score,
	// or an optional map with "true" and "false" descriptions for Noul.
	Criteria any `json:"criteria,omitempty"`
}

// DecisionRequest evaluates named questions against the same state.
type DecisionRequest struct {
	// State may be text, an object, or an array.
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
	// Model overrides the provider's configured model when set.
	Model string `json:"model,omitempty"`
}

// Answer contains the value for its Type. Noul is a probability in [0,1];
// Choice is an option name; Score is a weighted level index.
type Answer struct {
	Type          QuestionType       `json:"type"`
	Noul          float64            `json:"noul,omitempty"`
	Choice        string             `json:"choice,omitempty"`
	Score         float64            `json:"score,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	// Confidence is returned for Choice and Score, not Noul.
	Confidence float64           `json:"confidence,omitempty"`
	Legend     map[string]string `json:"legend,omitempty"`
}

// DecisionResponse holds one answer per question.
type DecisionResponse struct {
	Model   string            `json:"model"`
	Answers map[string]Answer `json:"answers"`
	Usage   Usage             `json:"usage"`
}

// NewDeciderFunc constructs a decision provider.
type NewDeciderFunc func(...Option) Decider

var deciders = make(map[string]NewDeciderFunc)

// RegisterDecider registers a decision provider independently of chat providers.
func RegisterDecider(name string, fn NewDeciderFunc) { deciders[name] = fn }

// NewDecider returns the named provider, or nil if it is not registered.
func NewDecider(name string, opts ...Option) Decider {
	if fn, ok := deciders[name]; ok {
		return fn(opts...)
	}
	return nil
}
