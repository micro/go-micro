package chat

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v6/agent"
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
	dir, _ := os.Getwd()
	return store.Scope(s.state, "chat", fmt.Sprintf("%x", sha256.Sum256([]byte(dir))))
}

func (s *session) openSession(id string, fresh bool) error {
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
func (s *session) showHistory() error {
	if len(s.agents) > 0 {
		return nil
	}
	messages, err := agent.LoadHistory(s.state, "micro-chat", s.id)
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
