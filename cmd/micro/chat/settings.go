package chat

import (
	"fmt"
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
	if provider == "" {
		provider = model.AutoDetectProvider(baseURL)
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
	if apiKey == "" && readline.IsTerminal(int(os.Stdin.Fd())) {
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
		saved.APIKey = apiKey
	} else if provider != saved.Provider || baseURL != saved.BaseURL {
		saved.APIKey = ""
	}
	if apiKey == "" {
		return fmt.Errorf("no API key configured; set --api_key or %s", envVarForProvider(provider))
	}
	configured := model.New(provider, model.WithAPIKey(apiKey), model.WithModel(modelName), model.WithBaseURL(baseURL))
	if configured == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	s.provider, s.modelName, s.baseURL, s.apiKey = provider, configured.Options().Model, baseURL, apiKey
	saved.Provider, saved.Model, saved.BaseURL = provider, s.modelName, baseURL
	return s.conversations().Write(store.NewRecord("settings", saved))
}
