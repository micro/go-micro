package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

// SessionInfo identifies a stored conversation. The catalog is derived from run
// records and includes conversations started through Go, RPC, or the CLI.
type SessionInfo struct {
	ID      string    `json:"id"`
	Updated time.Time `json:"updated"`
}

func ListSessions(s store.Store, name string) ([]SessionInfo, error) {
	runs, err := ListRunSummaries(s, name)
	if err != nil {
		return nil, err
	}
	sessions := map[string]time.Time{}
	for _, run := range runs {
		if run.UpdatedAt.After(sessions[run.Session]) {
			sessions[run.Session] = run.UpdatedAt
		}
	}
	result := make([]SessionInfo, 0, len(sessions))
	for id, updated := range sessions {
		result = append(result, SessionInfo{ID: id, Updated: updated})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Updated.Equal(result[j].Updated) {
			return result[i].ID < result[j].ID
		}
		return result[i].Updated.After(result[j].Updated)
	})
	return result, nil
}

type sessionHandler struct{ agent *agentImpl }

func (h *sessionHandler) List(ctx context.Context, _ *pb.ListSessionsRequest, rsp *pb.ListSessionsResponse) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sessions, err := ListSessions(h.agent.opts.Store, h.agent.opts.Name)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		rsp.Sessions = append(rsp.Sessions, &pb.Session{Id: session.ID, Updated: session.Updated.Unix()})
	}
	return nil
}

func (h *sessionHandler) History(ctx context.Context, req *pb.HistoryRequest, rsp *pb.HistoryResponse) error {
	a, release, err := h.agent.rpcSession(WithSession(ctx, req.Session))
	if err != nil {
		return err
	}
	defer release()
	a.mu.Lock()
	defer a.mu.Unlock()
	var messages []model.Message
	if a.opts.Memory == nil {
		if req.Query != "" {
			messages, err = SearchHistory(a.opts.Store, a.opts.Name, req.Session, req.Query, int(req.Limit))
		} else {
			messages, err = LoadHistory(a.opts.Store, a.opts.Name, req.Session)
		}
		if err != nil {
			return err
		}
	} else {
		messages = a.opts.Memory.Messages()
		if req.Query != "" {
			if recall, ok := a.opts.Memory.(MemoryRecall); ok {
				messages = append(recall.Recall(req.Query, int(req.Limit)), messages...)
			}
			messages = searchMessages(messages, req.Query, int(req.Limit))
		}
	}
	for _, msg := range messages {
		content, err := json.Marshal(msg.Content)
		if err != nil {
			return err
		}
		rsp.Messages = append(rsp.Messages, &pb.Message{Role: msg.Role, Content: fmt.Sprint(msg.Content), ContentJson: content})
	}
	return nil
}

func (h *sessionHandler) Compact(ctx context.Context, req *pb.CompactRequest, rsp *pb.HistoryResponse) error {
	a, release, err := h.agent.rpcSession(WithSession(ctx, req.Session))
	if err != nil {
		return err
	}
	defer release()
	if err := a.CompactHistory(int(req.KeepRecent)); err != nil {
		return err
	}
	for _, message := range a.mem.Messages() {
		content, err := json.Marshal(message.Content)
		if err != nil {
			return err
		}
		rsp.Messages = append(rsp.Messages, &pb.Message{Role: message.Role, Content: fmt.Sprint(message.Content), ContentJson: content})
	}
	return nil
}
