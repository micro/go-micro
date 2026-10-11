package model

import "strings"

// APIKeyEnv returns the conventional API key variable for a provider.
// Providers without a dedicated variable use MICRO_AI_API_KEY.
func APIKeyEnv(provider string) string {
	switch provider {
	case "openai", "anthropic", "gemini", "groq", "mistral", "together", "minimax", "ollama", "atlascloud":
		return strings.ToUpper(provider) + "_API_KEY"
	default:
		return "MICRO_AI_API_KEY"
	}
}
