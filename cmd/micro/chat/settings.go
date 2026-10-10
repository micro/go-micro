package chat

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

type settings struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key,omitempty"`
}

func (s *session) configure(c *cli.Context) error {
	var saved settings
	records, err := s.conversations().Read("settings")
	if err != nil && err != store.ErrNotFound {
		return err
	}
	if len(records) > 0 {
		if err := records[0].Decode(&saved); err != nil {
			return err
		}
	}
	provider := c.String("provider")
	if provider == "" {
		provider = saved.Provider
	}
	baseURL := c.String("base_url")
	if baseURL == "" && (provider == saved.Provider || provider == "") {
		baseURL = saved.BaseURL
	}
	interactive := c.String("prompt") == "" && c.String("host") == "" && readline.IsTerminal(int(os.Stdin.Fd()))
	if provider == "" && baseURL == "" && c.String("api_key") == "" {
		providers := configuredProviders()
		if len(providers) == 1 {
			provider = providers[0]
		} else if interactive {
			var err error
			provider, err = setupProvider()
			if err != nil {
				return err
			}
		} else if len(providers) > 1 {
			return fmt.Errorf("multiple provider keys configured; select one with --provider")
		}
	}
	if provider == "" {
		provider = model.AutoDetectProvider(baseURL)
	}
	if !model.ProviderCapabilities(provider).Model {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	modelName := c.String("model")
	if modelName == "" && provider == saved.Provider && baseURL == saved.BaseURL {
		modelName = saved.Model
	}
	apiKey := c.String("api_key")
	if apiKey == "" {
		apiKey = fallbackAPIKey(provider)
	}
	if apiKey == "" && provider == saved.Provider && baseURL == saved.BaseURL {
		apiKey = saved.APIKey
	}
	if apiKey == "" && provider != "ollama" && interactive {
		terminal, err := readline.NewEx(&readline.Config{Prompt: "", HistoryLimit: -1})
		if err != nil {
			return err
		}
		defer terminal.Close()
		fmt.Fprintf(terminal.Stdout(), "Configure %s. The key you enter is saved locally for this project.\n", provider)
		key, err := terminal.ReadPassword("API key: ")
		if err != nil {
			return err
		}
		apiKey = strings.TrimSpace(string(key))
		if apiKey == "" {
			return io.EOF
		}
		saved.APIKey = apiKey
	} else if provider != saved.Provider || baseURL != saved.BaseURL {
		saved.APIKey = ""
	}
	if apiKey == "" && provider != "ollama" {
		return fmt.Errorf("no API key configured; set --api_key or %s", envVarForProvider(provider))
	}
	configured := model.New(provider, model.WithAPIKey(apiKey), model.WithModel(modelName), model.WithBaseURL(baseURL))
	if configured == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	s.provider, s.modelName, s.baseURL, s.apiKey = provider, configured.Options().Model, baseURL, apiKey
	saved.Provider, saved.Model, saved.BaseURL = provider, s.modelName, baseURL
	if err := s.conversations().Write(store.NewRecord(profileKey(provider, baseURL), saved)); err != nil {
		return err
	}
	return s.conversations().Write(store.NewRecord("settings", saved))
}

// Only provider-specific keys identify a provider. MICRO_AI_API_KEY needs a
// provider flag (or the OpenAI default), not a guess based on registration order.
func configuredProviders() []string {
	var providers []string
	for _, provider := range model.RegisteredProviders("model") {
		name := envVarForProvider(provider)
		if name != "MICRO_AI_API_KEY" && os.Getenv(name) != "" {
			providers = append(providers, provider)
		}
	}
	return providers
}

func setupProvider() (string, error) {
	terminal, err := readline.NewEx(&readline.Config{Prompt: "Provider: ", HistoryLimit: -1})
	if err != nil {
		return "", err
	}
	defer terminal.Close()
	fmt.Fprintln(terminal.Stdout(), "Welcome to micro chat. Choose a model provider to get started.")
	for {
		selected, err := choose(terminal, "Providers", model.RegisteredProviders("model"), "none")
		if err != nil {
			return "", err
		}
		if selected == "" {
			return "", io.EOF
		}
		if model.ProviderCapabilities(selected).Model {
			return selected, nil
		}
		fmt.Fprintf(terminal.Stdout(), "Unknown provider %q. Choose a provider from the list.\n", selected)
	}
}
