package chat

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v6/agent"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/store"
)

type conversation struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Updated time.Time `json:"updated"`
}

func (s *session) writer() io.Writer {
	if s.output != nil {
		return s.output
	}
	return os.Stdout
}

func (s *session) conversations() store.Store {
	dir := s.project
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return store.Scope(s.state, "chat", fmt.Sprintf("%x", sha256.Sum256([]byte(dir))))
}

func (s *session) openSession(id string, fresh bool) error {
	var err error
	s.project, err = os.Getwd()
	if err != nil {
		return err
	}
	s.state = store.NewFileStore()
	if fresh {
		id = uuid.NewString()
	}
	if id == "" {
		records, err := s.conversations().Read("current")
		if err != nil && err != store.ErrNotFound {
			_ = s.state.Close()
			return err
		}
		if len(records) > 0 {
			id = string(records[0].Value)
		}
	}
	if id == "" {
		id = uuid.NewString()
	}
	if err := s.selectSession(id); err != nil {
		_ = s.state.Close()
		return err
	}
	return nil
}

func (s *session) selectSession(id string) error {
	if len(id) > 128 {
		return fmt.Errorf("session ID exceeds 128 bytes")
	}
	if err := s.conversations().Write(&store.Record{Key: "current", Value: []byte(id)}); err != nil {
		return err
	}
	s.id = id
	s.reset()
	return nil
}

func (s *session) remember(prompt string) error {
	key := "session/" + s.id
	value := conversation{ID: s.id, Title: prompt, Updated: time.Now()}
	if len(value.Title) > 80 {
		value.Title = string([]rune(prompt)[:min(80, len([]rune(prompt)))])
	}
	records, err := s.conversations().Read(key)
	if err != nil && err != store.ErrNotFound {
		return err
	}
	if len(records) > 0 {
		var previous conversation
		if err := records[0].Decode(&previous); err != nil {
			return err
		}
		value.Title = previous.Title
	}
	return s.conversations().Write(store.NewRecord(key, value))
}

// showHistory uses the framework's stored conversation. Remote history is owned
// by the remote agent and cannot be read from this client's local store.
func (s *session) showHistoryContext(ctx context.Context) error {
	if len(s.agents) > 0 {
		for name, info := range s.agents {
			if !info.History {
				fmt.Fprintf(s.writer(), "%s does not advertise remote history.\n", name)
				continue
			}
			response, err := pb.NewAgentSessionsService(name, s.cl).History(ctx, &pb.HistoryRequest{Session: s.id})
			if err != nil {
				return err
			}
			fmt.Fprintf(s.writer(), "%s / %s\n", name, s.id)
			for _, message := range response.Messages {
				fmt.Fprintf(s.writer(), "%s: %s\n", message.Role, message.Content)
			}
		}
		return nil
	}
	messages, err := agent.LoadHistory(s.state, "micro-chat", s.localSessionID())
	if err != nil {
		return err
	}
	for _, message := range messages {
		if (message.Role == "user" || message.Role == "assistant") && message.Content != "" {
			fmt.Fprintf(s.writer(), "%s: %s\n", message.Role, message.Content)
		}
	}
	return nil
}

// localSessionID keeps an explicit conversation name private to its project.
// The displayed ID and remote RPC session IDs retain their original values.
func (s *session) localSessionID() string {
	return s.projectSessionID(s.id)
}

func (s *session) projectSessionID(id string) string {
	dir := s.project
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(dir+"\x00"+id)))
}

func (s *session) remoteSessions(ctx context.Context) error {
	for name, info := range s.agents {
		if !info.History {
			return fmt.Errorf("%s does not advertise a session catalog", name)
		}
		response, err := pb.NewAgentSessionsService(name, s.cl).List(ctx, &pb.ListSessionsRequest{})
		if err != nil {
			return err
		}
		for _, session := range response.Sessions {
			fmt.Fprintf(s.writer(), "%s  %s  %s\n", name, session.Id, time.Unix(session.Updated, 0).Format(time.RFC3339))
		}
	}
	return nil
}
