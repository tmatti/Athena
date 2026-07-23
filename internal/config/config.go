package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

const minAPIKeyLength = 16

type Config struct {
	DatabaseURL         string `env:"DATABASE_URL,required"`
	BrainAPIKey         string `env:"BRAIN_API_KEY,required"`
	Port                int    `env:"PORT" envDefault:"8080"`
	EmbeddingProvider   string `env:"EMBEDDING_PROVIDER" envDefault:"openai_compatible"`
	EmbeddingBaseURL    string `env:"EMBEDDING_BASE_URL" envDefault:"https://openrouter.ai/api/v1"`
	EmbeddingModel      string `env:"EMBEDDING_MODEL" envDefault:"openai/text-embedding-3-small"`
	EmbeddingDimensions int    `env:"EMBEDDING_DIMENSIONS" envDefault:"1536"`
	EmbeddingAPIKey     string `env:"EMBEDDING_API_KEY"`
	LogLevel            string `env:"LOG_LEVEL" envDefault:"info"`
	// PublicBaseURL is the server's public origin (e.g.
	// https://athena.example.com). Setting it enables OAuth for MCP clients;
	// unset leaves the server exactly as before (static bearer key only).
	PublicBaseURL string `env:"PUBLIC_BASE_URL"`
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	if len(cfg.BrainAPIKey) < minAPIKeyLength {
		return Config{}, fmt.Errorf("BRAIN_API_KEY must be at least %d characters", minAPIKeyLength)
	}
	switch cfg.EmbeddingProvider {
	case "openai_compatible", "none":
	default:
		return Config{}, fmt.Errorf("EMBEDDING_PROVIDER must be \"openai_compatible\" or \"none\", got %q", cfg.EmbeddingProvider)
	}
	if cfg.EmbeddingDimensions < 1 {
		return Config{}, fmt.Errorf("EMBEDDING_DIMENSIONS must be positive, got %d", cfg.EmbeddingDimensions)
	}
	if cfg.PublicBaseURL != "" {
		normalized, err := validatePublicBaseURL(cfg.PublicBaseURL)
		if err != nil {
			return Config{}, err
		}
		cfg.PublicBaseURL = normalized
	}
	return cfg, nil
}

// validatePublicBaseURL requires an absolute origin with no path. OAuth token
// security depends on TLS, so plain http is only allowed on loopback hosts
// for local development.
func validatePublicBaseURL(raw string) (string, error) {
	raw = strings.TrimRight(raw, "/")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must be an absolute URL, got %q", raw)
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must not include a path, got %q", raw)
	}
	switch u.Scheme {
	case "https":
	case "http":
		h := u.Hostname()
		if h != "localhost" && h != "127.0.0.1" && h != "::1" {
			return "", fmt.Errorf("PUBLIC_BASE_URL must use https (http is allowed only on localhost), got %q", raw)
		}
	default:
		return "", fmt.Errorf("PUBLIC_BASE_URL must use https, got %q", raw)
	}
	return raw, nil
}
