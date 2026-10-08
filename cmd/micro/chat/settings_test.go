package chat

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/store"
)

func TestSavedSettingsDoNotSendKeyToDifferentEndpoint(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("MICRO_AI_API_KEY", "")
	s := &session{state: store.NewMemoryStore()}
	saved := settings{Provider: "openai", Model: "test-model", BaseURL: "http://first.invalid", APIKey: "saved-test-key"}
	if err := s.conversations().Write(store.NewRecord("settings", saved)); err != nil {
		t.Fatal(err)
	}
	flags := flag.NewFlagSet("chat", flag.ContinueOnError)
	flags.String("base_url", "http://second.invalid", "")
	ctx := cli.NewContext(cli.NewApp(), flags, nil)
	if err := s.configure(ctx); err == nil {
		t.Fatal("reused credentials for a different endpoint")
	}
}

func TestSavedSettingsRestoreModel(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("MICRO_AI_API_KEY", "")
	s := &session{state: store.NewMemoryStore()}
	saved := settings{Provider: "openai", Model: "test-model", APIKey: "saved-test-key"}
	if err := s.conversations().Write(store.NewRecord("settings", saved)); err != nil {
		t.Fatal(err)
	}
	ctx := cli.NewContext(cli.NewApp(), flag.NewFlagSet("chat", flag.ContinueOnError), nil)
	if err := s.configure(ctx); err != nil {
		t.Fatal(err)
	}
	if s.provider != "openai" || s.modelName != "test-model" || s.apiKey != "saved-test-key" {
		t.Fatal("settings not restored")
	}
}
