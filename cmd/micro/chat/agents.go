package chat

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"go-micro.dev/v6/store"
)

func (s *session) agentName() string {
	for name := range s.agents {
		return name
	}
	return "micro"
}

// Discovery does not change the current conversation or its destination.
func (s *session) availableAgents() map[string]agentInfo {
	discovery := &session{reg: s.reg}
	discovery.discoverAgents()
	return discovery.agents
}

func (s *session) selectAgent(name string) error {
	if name == "micro" {
		s.agents = nil
		return nil
	}
	info, ok := s.availableAgents()[name]
	if !ok {
		return fmt.Errorf("agent %q is not registered; use /agents to list available agents", name)
	}
	if !info.Sessions {
		return fmt.Errorf("agent %q does not support isolated conversations; rebuild it with the current Go Micro version", name)
	}
	s.agents = map[string]agentInfo{name: info}
	return nil
}

func (s *session) listAgents() {
	fmt.Fprintln(s.writer(), "micro  built-in development agent")
	available := s.availableAgents()
	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info := available[name]
		fmt.Fprintf(s.writer(), "%s  %s / %s\n", terminalText(name), terminalText(info.Provider), terminalText(info.Model))
	}
	fmt.Fprintf(s.writer(), "Current: %s. Use /agent NAME to switch.\n", terminalText(s.agentName()))
}

func (s *session) switchAgent(name string) error {
	if name == s.agentName() {
		return nil
	}
	previous := s.agents
	if err := s.selectAgent(name); err != nil {
		return err
	}
	rollback := func(err error) error { s.agents = previous; return err }
	if name == "micro" && s.configureLocal != nil {
		if err := s.configureLocal(); err != nil {
			return rollback(err)
		}
	}
	records, err := s.conversations().Read("current")
	if err != nil && err != store.ErrNotFound {
		return rollback(err)
	}
	id := uuid.NewString()
	if len(records) > 0 {
		id = string(records[0].Value)
	}
	if err := s.selectSession(id); err != nil {
		return rollback(err)
	}
	s.refreshTools()
	return nil
}
