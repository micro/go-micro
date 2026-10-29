package chat

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

func registeredSupport(t *testing.T) registry.Registry {
	t.Helper()
	reg := registry.NewMemoryRegistry()
	if err := reg.Register(&registry.Service{Name: "support", Metadata: map[string]string{"agent": "v1"}, Nodes: []*registry.Node{{Id: "support-1", Address: "127.0.0.1:1234", Metadata: map[string]string{"sessions": "v1"}}}}); err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestDefaultChatDoesNotSelectRegisteredAgent(t *testing.T) {
	oldDir := store.DefaultDir
	store.DefaultDir = t.TempDir()
	defer func() { store.DefaultDir = oldDir }()
	options := cmd.DefaultOptions()
	oldReg, oldClient := *options.Registry, *options.Client
	remote := &remoteClient{Client: oldClient}
	*options.Registry, *options.Client = registeredSupport(t), remote
	defer func() { *options.Registry, *options.Client = oldReg, oldClient }()
	flags := flag.NewFlagSet("chat", flag.ContinueOnError)
	flags.String("provider", "unconfigured", "")
	flags.String("prompt", "hello", "")
	// The local configuration error must not be bypassed by a registered agent.
	err := run(cli.NewContext(cli.NewApp(), flags, nil))
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("expected local setup, got %v", err)
	}
	if remote.calls != 0 {
		t.Fatalf("silently called remote agent %d times", remote.calls)
	}
}

func TestAgentSwitchKeepsConversationsSeparate(t *testing.T) {
	var out bytes.Buffer
	s := &session{reg: registeredSupport(t), state: store.NewMemoryStore(), project: t.TempDir(), output: &out}
	if err := s.selectSession("local-thread"); err != nil {
		t.Fatal(err)
	}
	if err := s.remember("local question"); err != nil {
		t.Fatal(err)
	}
	s.listAgents()
	if s.agentName() != "micro" || !strings.Contains(out.String(), "support") {
		t.Fatal("listing changed routing or omitted support")
	}
	if err := s.switchAgent("support"); err != nil {
		t.Fatal(err)
	}
	if s.id == "local-thread" {
		t.Fatal("remote reused local conversation")
	}
	remoteID := s.id
	if err := s.remember("support question"); err != nil {
		t.Fatal(err)
	}
	if err := s.switchAgent("missing"); err == nil {
		t.Fatal("unknown agent accepted")
	}
	if s.agentName() != "support" || s.id != remoteID {
		t.Fatal("failed switch changed conversation")
	}
	if err := s.switchAgent("micro"); err != nil {
		t.Fatal(err)
	}
	if s.id != "local-thread" {
		t.Fatalf("local conversation lost: %s", s.id)
	}
	records, err := s.conversations().Read("session/local-thread")
	if err != nil {
		t.Fatal(err)
	}
	var entry conversation
	if err := records[0].Decode(&entry); err != nil {
		t.Fatal(err)
	}
	if entry.Title != "local question" {
		t.Fatal("remote title leaked into local history")
	}
	if err := s.switchAgent("support"); err != nil {
		t.Fatal(err)
	}
	if s.id != remoteID {
		t.Fatal("remote conversation not restored")
	}
}

func TestRemoteConversationIsScopedToProject(t *testing.T) {
	state := store.NewMemoryStore()
	first := &session{state: state, project: t.TempDir(), agents: map[string]agentInfo{"support": {}}}
	second := &session{state: state, project: t.TempDir(), agents: map[string]agentInfo{"support": {}}}
	if err := first.selectSession("first-project"); err != nil {
		t.Fatal(err)
	}
	if records, err := second.conversations().Read("current"); err == nil && len(records) != 0 {
		t.Fatal("remote conversation leaked between projects")
	}
	if records, err := store.Scope(state, "agent", "support").Read("current"); err == nil && len(records) != 0 {
		t.Fatal("CLI state leaked into agent's store")
	}
}
