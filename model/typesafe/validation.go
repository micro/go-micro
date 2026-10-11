package typesafe

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"go-micro.dev/v6/model"
)

// Validate the JSON representation so callers can use structs, maps, or slices.
func validateRequest(data []byte) error {
	var req struct {
		State     json.RawMessage
		Questions map[string]struct {
			Type         model.QuestionType
			Instructions json.RawMessage
			Criteria     json.RawMessage
		}
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	if !content(req.State) {
		return fmt.Errorf("typesafe: state must be text, an object, or an array")
	}
	for name, q := range req.Questions {
		if !content(q.Instructions) {
			return fmt.Errorf("typesafe: question %q requires instructions", name)
		}
		switch q.Type {
		case model.Noul, model.Choice:
			var criteria map[string]json.RawMessage
			if len(q.Criteria) > 0 {
				if err := json.Unmarshal(q.Criteria, &criteria); err != nil {
					return fmt.Errorf("typesafe: question %q requires a criteria object", name)
				}
			}
			if q.Type == model.Choice && (len(criteria) == 0 || len(criteria) > 255) {
				return fmt.Errorf("typesafe: question %q requires 1–255 choices", name)
			}
			for key, value := range criteria {
				if q.Type == model.Noul && key != "true" && key != "false" {
					return fmt.Errorf("typesafe: question %q has invalid yes/no criteria", name)
				}
				if !content(value) && (q.Type != model.Choice || string(value) != "null") {
					return fmt.Errorf("typesafe: question %q has invalid criteria", name)
				}
			}
		case model.Score:
			var levels []json.RawMessage
			if err := json.Unmarshal(q.Criteria, &levels); err != nil || len(levels) < 2 || len(levels) > 10 {
				return fmt.Errorf("typesafe: question %q requires 2–10 score levels", name)
			}
			for _, value := range levels {
				if !content(value) {
					return fmt.Errorf("typesafe: question %q has invalid score criteria", name)
				}
			}
		default:
			return fmt.Errorf("typesafe: question %q has unsupported type %q", name, q.Type)
		}
	}
	return nil
}

func content(data json.RawMessage) bool {
	return len(data) > 0 && (data[0] == '"' || data[0] == '{' || data[0] == '[')
}

func decodeResponse(data []byte, questions map[string]model.Question) (*model.DecisionResponse, error) {
	var response model.DecisionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("typesafe: decode response: %w", err)
	}
	var raw struct {
		Answers map[string]map[string]json.RawMessage
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("typesafe: decode answers: %w", err)
	}
	if len(response.Answers) != len(questions) {
		return nil, fmt.Errorf("typesafe: response does not answer every question")
	}
	for name, question := range questions {
		answer, ok := response.Answers[name]
		if !ok || answer.Type != question.Type {
			return nil, fmt.Errorf("typesafe: missing or mismatched answer for %q", name)
		}
		required := []string{string(question.Type)}
		if question.Type != model.Noul {
			required = append(required, "probabilities", "confidence")
		}
		if question.Type == model.Score {
			required = append(required, "legend")
		}
		for _, field := range required {
			value := raw.Answers[name][field]
			if len(value) == 0 || string(value) == "null" {
				return nil, fmt.Errorf("typesafe: answer %q is missing %s", name, field)
			}
		}
		if question.Type == model.Noul {
			if !probability(answer.Noul) {
				return nil, fmt.Errorf("typesafe: answer %q has invalid probability", name)
			}
			continue
		}
		criteria, err := json.Marshal(question.Criteria)
		if err != nil {
			return nil, fmt.Errorf("typesafe: encode criteria: %w", err)
		}
		options := map[string]json.RawMessage{}
		if question.Type == model.Choice {
			if err := json.Unmarshal(criteria, &options); err != nil {
				return nil, err
			}
			if _, ok := options[answer.Choice]; !ok {
				return nil, fmt.Errorf("typesafe: answer %q chose an unknown option", name)
			}
		} else {
			var levels []json.RawMessage
			if err := json.Unmarshal(criteria, &levels); err != nil {
				return nil, err
			}
			for i := range levels {
				options[strconv.Itoa(i)] = nil
			}
			if answer.Score < 0 || answer.Score > float64(len(levels)-1) || len(answer.Legend) != len(levels) {
				return nil, fmt.Errorf("typesafe: answer %q has invalid score or legend", name)
			}
			for key := range options {
				if _, ok := answer.Legend[key]; !ok {
					return nil, fmt.Errorf("typesafe: answer %q has invalid legend", name)
				}
			}
		}
		if len(answer.Probabilities) != len(options) || !probability(answer.Confidence) {
			return nil, fmt.Errorf("typesafe: answer %q has invalid probabilities or confidence", name)
		}
		var probabilities map[string]*float64
		if err := json.Unmarshal(raw.Answers[name]["probabilities"], &probabilities); err != nil {
			return nil, fmt.Errorf("typesafe: answer %q has invalid probabilities: %w", name, err)
		}
		var sum float64
		for key := range options {
			p := probabilities[key]
			if p == nil || !probability(*p) {
				return nil, fmt.Errorf("typesafe: answer %q has invalid probability for %q", name, key)
			}
			sum += *p
		}
		if math.Abs(sum-1) > 0.001 {
			return nil, fmt.Errorf("typesafe: answer %q probabilities do not sum to one", name)
		}
	}
	response.Usage.TotalTokens = response.Usage.InputTokens + response.Usage.OutputTokens
	return &response, nil
}

func probability(value float64) bool { return value >= 0 && value <= 1 }
