package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

// HistoryMatch identifies the conversation a recalled message came from.
type HistoryMatch struct {
	Session string        `json:"session"`
	Message model.Message `json:"message"`
}

// SearchSessions searches retained and archived messages in explicitly allowed
// conversations belonging to one agent. The caller must authorize the session
// IDs; this function never discovers or expands that scope. Results are ranked
// by matching query words, then input session order and newest message first.
// Only user and assistant messages are returned. No model calls are made.
func SearchSessions(ctx context.Context, s store.Store, name string, ids []string, query string, limit int) ([]HistoryMatch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	terms := recallTerms(query)
	if len(terms) == 0 {
		terms = strings.Fields(strings.ToLower(query))
	}
	if len(terms) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	limit = min(limit, 100)
	type match struct {
		HistoryMatch
		score int
	}
	var matches []match
	sessions := map[string]bool{}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if sessions[id] {
			continue
		}
		sessions[id] = true
		state, err := loadHistoryState(s, name, id)
		if err != nil {
			return nil, err
		}
		messages := append(state.Archive, state.Messages...)
		seen := map[string]bool{}
		for i := len(messages) - 1; i >= 0; i-- {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			message := messages[i]
			if message.Role != "user" && message.Role != "assistant" {
				continue
			}
			score := recallScore(message, terms)
			key := message.Role + "\x00" + fmt.Sprint(message.Content)
			if score == 0 || seen[key] {
				continue
			}
			seen[key] = true
			matches = append(matches, match{HistoryMatch: HistoryMatch{Session: id, Message: message}, score: score})
			sort.SliceStable(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
			if len(matches) > limit {
				matches = matches[:limit]
			}
		}
	}
	result := make([]HistoryMatch, len(matches))
	for i, match := range matches {
		result[i] = match.HistoryMatch
	}
	return result, nil
}
