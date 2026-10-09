package chat

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func (s *session) currentModel() model.Model {
	return model.New(s.provider, model.WithModel(s.modelName), model.WithAPIKey(s.apiKey), model.WithBaseURL(s.baseURL))
}

func profileKey(provider, endpoint string) string {
	return fmt.Sprintf("providers/%x", sha256.Sum256([]byte(provider+"\x00"+endpoint)))
}

func (s *session) saveModel(next settings, key string) error {
	if !model.ProviderCapabilities(next.Provider).Model {
		return fmt.Errorf("unknown provider: %s", next.Provider)
	}
	configured := model.New(next.Provider, model.WithModel(next.Model), model.WithBaseURL(next.BaseURL), model.WithAPIKey(key))
	if configured == nil {
		return fmt.Errorf("unknown provider: %s", next.Provider)
	}
	next.Model = configured.Options().Model
	if err := s.conversations().Write(store.NewRecord(profileKey(next.Provider, next.BaseURL), next)); err != nil {
		return err
	}
	if err := s.conversations().Write(store.NewRecord("settings", next)); err != nil {
		return err
	}
	s.provider, s.modelName, s.baseURL, s.apiKey = next.Provider, next.Model, next.BaseURL, key
	s.reset()
	return nil
}

func (s *session) selectModel(name string) error {
	var saved settings
	records, err := s.conversations().Read("settings")
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if len(records) > 0 {
		if err := records[0].Decode(&saved); err != nil {
			return err
		}
	}
	next := settings{Provider: s.provider, Model: name, BaseURL: s.baseURL}
	if saved.Provider == s.provider && saved.BaseURL == s.baseURL {
		next.APIKey = saved.APIKey
	}
	return s.saveModel(next, s.apiKey)
}

func choose(terminal *readline.Instance, title string, choices []string, current string) (string, error) {
	fmt.Fprintf(terminal.Stdout(), "%s (current: %s)\n", title, current)
	for i, name := range choices {
		fmt.Fprintf(terminal.Stdout(), "  %d. %s\n", i+1, name)
	}
	fmt.Fprintln(terminal.Stdout(), "Enter a number or name; empty input cancels.")
	line, err := terminal.Readline()
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if number, err := strconv.Atoi(line); err == nil {
		if number < 1 || number > len(choices) {
			return "", fmt.Errorf("selection out of range")
		}
		return choices[number-1], nil
	}
	return line, nil
}

func (s *session) modelCommand(ctx context.Context, terminal *readline.Instance, line string) error {
	if len(s.agents) > 0 {
		for name, info := range s.agents {
			provider, selected := info.Provider, info.Model
			if provider == "" {
				provider = "not advertised"
			}
			if selected == "" {
				selected = "not advertised"
			}
			fmt.Fprintf(s.writer(), "%s: provider=%s model=%s\n", name, provider, selected)
		}
		return fmt.Errorf("connected agents own their model configuration; configure the agent on its host")
	}
	command, arg, _ := strings.Cut(line, " ")
	arg = strings.TrimSpace(arg)
	switch command {
	case "/models", "/model":
		name := arg
		if command == "/models" || name == "" {
			names, err := model.ListModels(ctx, s.currentModel())
			if err != nil {
				return fmt.Errorf("%s; select a model directly with /model ID", err)
			}
			if command == "/models" {
				for _, id := range names {
					fmt.Fprintln(s.writer(), id)
				}
				return nil
			}
			name, err = choose(terminal, "Models", names, s.modelName)
			if err != nil {
				return err
			}
		}
		if name == "" {
			return nil
		}
		if err := s.selectModel(name); err != nil {
			return err
		}
	case "/provider":
		provider, endpoint, _ := strings.Cut(arg, " ")
		endpoint = strings.TrimSpace(endpoint)
		if provider == "" {
			var err error
			provider, err = choose(terminal, "Providers", model.RegisteredProviders("model"), s.provider)
			if err != nil {
				return err
			}
		}
		if provider == "" {
			return nil
		}
		if !model.ProviderCapabilities(provider).Model {
			return fmt.Errorf("unknown provider: %s", provider)
		}
		if provider == s.provider && endpoint == "" {
			endpoint = s.baseURL
		}
		next := settings{Provider: provider, BaseURL: endpoint}
		records, err := s.conversations().Read(profileKey(provider, endpoint))
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return err
		}
		if len(records) > 0 {
			if err := records[0].Decode(&next); err != nil {
				return err
			}
		}
		key := fallbackAPIKey(provider)
		if provider == s.provider && endpoint == s.baseURL {
			key = s.apiKey
			next.Model = s.modelName
		}
		if key == "" {
			key = next.APIKey
		}
		if key == "" && provider != "ollama" {
			fmt.Fprintf(s.writer(), "Configure %s. Entered keys are saved locally for this project. Empty input cancels.\n", provider)
			entered, err := terminal.ReadPassword("API key: ")
			if err != nil {
				return err
			}
			key = strings.TrimSpace(string(entered))
			if key == "" {
				return nil
			}
			next.APIKey = key
		}
		if err := s.saveModel(next, key); err != nil {
			return err
		}
	}
	fmt.Fprintf(s.writer(), "Provider: %s · Model: %s\n", s.provider, s.modelName)
	return nil
}
