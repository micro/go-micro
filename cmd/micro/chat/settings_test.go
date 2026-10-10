package chat

import (
	"context"
	"flag"
	"go-micro.dev/v6/model"
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

func TestModelSwitchPreservesConversation(t *testing.T) {
	var observed []model.Message
	s := testSession(t, func(_ context.Context, r *model.Request, o model.Options) (*model.Response, error) {
		observed = r.Messages
		return &model.Response{Reply: o.Model}, nil
	})
	s.state = store.NewMemoryStore()
	s.id = "conversation"
	if err := s.conversations().Write(store.NewRecord("settings", settings{Provider: s.provider, Model: s.modelName, BaseURL: s.baseURL, APIKey: "saved"})); err != nil {
		t.Fatal(err)
	}
	if _, err := s.developmentAgent().Ask(context.Background(), "remember this"); err != nil {
		t.Fatal(err)
	}
	if err := s.selectModel("replacement"); err != nil {
		t.Fatal(err)
	}
	response, err := s.developmentAgent().Ask(context.Background(), "continue")
	if err != nil || response.Reply != "replacement" || len(observed) != 2 {
		t.Fatalf("switch lost context: %+v %+v %v", response, observed, err)
	}
	records, _ := s.conversations().Read("settings")
	var saved settings
	records[0].Decode(&saved)
	if saved.APIKey != "saved" || saved.Model != "replacement" {
		t.Fatalf("saved settings changed: %+v", saved)
	}
}

func TestUnknownProviderDoesNotReplaceSettings(t *testing.T) {
	s := &session{state: store.NewMemoryStore(), provider: "openai", modelName: "original", apiKey: "key"}
	if err := s.saveModel(settings{Provider: "missing-provider"}, "other"); err == nil {
		t.Fatal("unknown provider accepted")
	}
	if s.provider != "openai" || s.modelName != "original" || s.apiKey != "key" {
		t.Fatal("failed switch changed configuration")
	}
}

func TestConfigureProviderFromEnvironment(t *testing.T) {
	for _, provider := range model.RegisteredProviders("model") {
		t.Setenv(envVarForProvider(provider), "")
	}
	t.Setenv("GROQ_API_KEY", "environment-key")
	s := &session{state: store.NewMemoryStore()}
	flags := flag.NewFlagSet("chat", flag.ContinueOnError)
	flags.String("prompt", "hello", "")
	ctx := cli.NewContext(cli.NewApp(), flags, nil)
	if err := s.configure(ctx); err != nil {
		t.Fatal(err)
	}
	if s.provider != "groq" {
		t.Fatalf("provider = %s", s.provider)
	}
	records, err := s.conversations().Read("settings")
	if err != nil {
		t.Fatal(err)
	}
	var saved settings
	if err := records[0].Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.APIKey != "" {
		t.Fatal("environment key copied to storage")
	}

	// More than one configured key must not choose a provider arbitrarily.
	t.Setenv("OPENAI_API_KEY", "other-environment-key")
	s = &session{state: store.NewMemoryStore()}
	if err := s.configure(ctx); err == nil {
		t.Fatal("ambiguous keys accepted")
	}
	flags.String("provider", "groq", "")
	if err := s.configure(ctx); err != nil {
		t.Fatal(err)
	}
	if s.provider != "groq" {
		t.Fatalf("explicit provider = %s", s.provider)
	}
}
