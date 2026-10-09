package chat

import (
	"context"
	"fmt"
	"strings"

	"go-micro.dev/v6/agent"
	pb "go-micro.dev/v6/agent/proto"
)

func (s *session) memoryCommand(ctx context.Context, line string) error {
	compact := line == "/compact"
	query := strings.TrimSpace(strings.TrimPrefix(line, "/search "))
	if !compact && (query == "--all" || strings.HasPrefix(query, "--all ")) {
		if len(s.agents) > 0 {
			return fmt.Errorf("cross-conversation search is local; hosted agents control their own recall scope")
		}
		results, err := s.searchConversations(ctx, strings.TrimSpace(strings.TrimPrefix(query, "--all")))
		if err != nil {
			return err
		}
		for _, result := range results {
			fmt.Fprintf(s.writer(), "%s · %s / %s: %s\n", terminalText(result.Session), terminalText(result.Title), result.Role, terminalText(result.Content))
		}
		if len(results) == 0 {
			fmt.Fprintln(s.writer(), "No matching messages in this project's recent conversations.")
		}
		return nil
	}
	if len(s.agents) > 0 {
		for name, info := range s.agents {
			if !info.History {
				return fmt.Errorf("%s does not advertise history controls", name)
			}
			api := pb.NewAgentSessionsService(name, s.cl)
			var response *pb.HistoryResponse
			var err error
			if compact {
				response, err = api.Compact(ctx, &pb.CompactRequest{Session: s.id, KeepRecent: 12})
			} else {
				response, err = api.History(ctx, &pb.HistoryRequest{Session: s.id, Query: query, Limit: 20})
			}
			if err != nil {
				return err
			}
			if compact {
				fmt.Fprintf(s.writer(), "%s: %d context messages retained; older messages remain searchable.\n", name, len(response.Messages))
			} else {
				for _, message := range response.Messages {
					fmt.Fprintf(s.writer(), "%s / %s: %s\n", name, message.Role, message.Content)
				}
			}
		}
		return nil
	}
	if compact {
		if err := agent.CompactHistory(s.developmentAgent(), 12); err != nil {
			return err
		}
		fmt.Fprintln(s.writer(), "Context compacted, keeping 12 recent messages when available. Older messages remain searchable.")
		return nil
	}
	messages, err := agent.SearchHistory(s.state, "micro-chat", s.localSessionID(), query, 20)
	if err != nil {
		return err
	}
	for _, message := range messages {
		fmt.Fprintf(s.writer(), "%s: %v\n", message.Role, message.Content)
	}
	if len(messages) == 0 {
		fmt.Fprintln(s.writer(), "No matching messages in this conversation.")
	}
	return nil
}
