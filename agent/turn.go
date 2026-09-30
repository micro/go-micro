package agent

import (
	"context"
	"fmt"
	"go-micro.dev/v6/model"
)

// turnExecution owns tool iteration. Provider retries apply to one model turn,
// never to a completed sequence of side effects.
type turnExecution struct {
	model.Model
	agent   *agentImpl
	turner  model.Turner
	handler model.ToolHandler
	policy  model.GeneratePolicy
}
type turnAdapter struct {
	model.Model
	turner model.Turner
}

func (m turnAdapter) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	return m.turner.Turn(ctx, req, opts...)
}

func (m *turnExecution) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	next := *req
	out := &model.Response{}
	var saved = m.agent.currentRun
	durable := m.agent.opts.StrictRecovery && saved != nil
	if durable && saved.Continuation != nil {
		next.Continuation = saved.Continuation
	}
	for round := 0; round < 64; round++ {
		var response *model.Response
		var err error
		if durable && saved.PendingTurn != nil {
			response = saved.PendingTurn
		} else {
			policy := m.policy
			if durable {
				maxAttempts := policy.MaxAttempts
				if maxAttempts <= 0 {
					maxAttempts = 1
				}
				if saved.ModelTurns >= 64 || saved.TurnAttempts >= maxAttempts {
					return nil, model.ErrLimit
				}
				policy.MaxAttempts = maxAttempts - saved.TurnAttempts
				policy.BeforeAttempt = func(ctx context.Context, _ int) error { saved.TurnAttempts++; return m.agent.saveRun(ctx, *saved) }
			}
			response, err = model.GenerateWithRetry(ctx, turnAdapter{m.Model, m.turner}, &next, policy, opts...)
			if err != nil {
				return nil, err
			}
			if durable && response != nil {
				saved.PendingTurn = response
				saved.TurnAttempts = 0
				saved.ModelTurns++
				if err := m.agent.saveRun(ctx, *saved); err != nil {
					return nil, err
				}
			}
		}
		if response == nil {
			return nil, fmt.Errorf("provider returned no turn result")
		}
		out.Usage.InputTokens += response.Usage.InputTokens
		out.Usage.OutputTokens += response.Usage.OutputTokens
		out.Usage.TotalTokens += response.Usage.TotalTokens
		out.StopReason = response.StopReason
		if len(response.ToolCalls) == 0 {
			out.Reply = response.Reply
			out.Answer = response.Answer
			return out, nil
		}
		if response.Continuation == nil {
			return nil, fmt.Errorf("provider returned tool calls without continuation")
		}
		continuation := *response.Continuation
		continuation.Results = nil
		for _, call := range response.ToolCalls {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			result := m.handler(ctx, call)
			result.ID = call.ID
			result.Name = call.Name
			call.Result = result.Content
			out.ToolCalls = append(out.ToolCalls, call)
			continuation.Results = append(continuation.Results, result)
			// A guardrail refusal stops execution rather than giving the model another
			// opportunity to repeat an action or claim completion.
			if result.Refused != "" {
				if result.Refused == model.RefusedLoop || result.Refused == model.RefusedMaxSteps {
					return nil, fmt.Errorf("%w: %s", model.ErrLimit, result.Content)
				}
				return nil, fmt.Errorf("agent tool %s refused (%s): %s", call.Name, result.Refused, result.Content)
			}
		}
		next.Continuation = &continuation
		if durable {
			saved.PendingTurn = nil
			saved.Continuation = &continuation
			if err := m.agent.saveRun(ctx, *saved); err != nil {
				return nil, err
			}
		}
	}
	return nil, model.ErrLimit
}

func (a *agentImpl) generate(ctx context.Context, req *model.Request, policy model.GeneratePolicy) (*model.Response, error) {
	if a.ownsTurns {
		return a.model.Generate(ctx, req)
	}
	if a.opts.StrictRecovery {
		return nil, fmt.Errorf("strict recovery requires a one-turn provider")
	}
	return model.GenerateWithRetry(ctx, a.model, req, policy)
}
