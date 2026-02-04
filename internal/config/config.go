package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port string

	// LLM API Keys
	AnthropicAPIKey string
	OpenAIAPIKey    string
	GeminiAPIKey    string

	// TTS
	ElevenLabsAPIKey string

	// GitHub
	GitHubToken string
	GitHubOwner string
	GitHubRepo  string

	// Google Cloud (for STT)
	GoogleProjectID string

	// App settings
	DefaultLLMProvider string
	DefaultTTSProvider string
	DefaultSTTProvider string
	RepoPath           string
	AllowedOrigin      string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		AnthropicAPIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		GeminiAPIKey:       os.Getenv("GEMINI_API_KEY"),
		ElevenLabsAPIKey:   os.Getenv("ELEVENLABS_API_KEY"),
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		GitHubOwner:        os.Getenv("GITHUB_OWNER"),
		GitHubRepo:         os.Getenv("GITHUB_REPO"),
		GoogleProjectID:    os.Getenv("GOOGLE_PROJECT_ID"),
		DefaultLLMProvider: getEnv("DEFAULT_LLM_PROVIDER", "anthropic"),
		DefaultTTSProvider: getEnv("DEFAULT_TTS_PROVIDER", "openai"),
		DefaultSTTProvider: getEnv("DEFAULT_STT_PROVIDER", "openai"),
		RepoPath:           getEnv("REPO_PATH", "/tmp/flyagi-repo"),
		AllowedOrigin:      getEnv("ALLOWED_ORIGIN", "*"),
	}

	if cfg.Port == "" {
		return nil, fmt.Errorf("PORT must not be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
