// Package nlp provides natural language processing for command parsing.
package nlp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raja-aiml/air/internal/engine"
)

// Provider defines the interface for LLM backends.
type Provider interface {
	// Name returns the provider identifier
	Name() string

	// Parse interprets natural language input and returns a command match
	Parse(ctx context.Context, input string, commands []*engine.Command) (*ParseResult, error)
}

// ParseResult contains the result of parsing natural language input.
type ParseResult struct {
	Command    string
	Parameters map[string]any
	Confidence float64
	Source     string // "embeddings", "anthropic", "openai"
	RawInput   string
}

// ProviderConfig holds configuration for LLM providers.
type ProviderConfig struct {
	Type      string        // "anthropic", "openai", "auto"
	APIKey    string        // API key (if empty, uses provider-specific env var)
	Model     string        // Model name (if empty, uses default)
	MaxTokens int           // Max tokens for response
	Timeout   time.Duration // Request timeout
}

// DefaultConfig returns a default provider configuration.
func DefaultConfig() ProviderConfig {
	return ProviderConfig{
		Type:      "auto",
		MaxTokens: 1024,
		Timeout:   30 * time.Second,
	}
}

// NewProvider creates the appropriate provider based on config.
func NewProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case "anthropic":
		return NewAnthropicProvider(cfg)
	case "openai":
		return NewOpenAIProvider(cfg)
	case "auto":
		return NewAutoProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

// AutoProvider tries available API keys to find a working provider.
type AutoProvider struct {
	provider Provider
}

// NewAutoProvider creates a provider by detecting available API keys.
func NewAutoProvider(cfg ProviderConfig) (*AutoProvider, error) {
	// Try Anthropic first
	if key := getAPIKey("ANTHROPIC_API_KEY", cfg.APIKey); key != "" {
		anthropicCfg := cfg
		anthropicCfg.APIKey = key
		p, err := NewAnthropicProvider(anthropicCfg)
		if err == nil {
			return &AutoProvider{provider: p}, nil
		}
	}

	// Fall back to OpenAI
	if key := getAPIKey("OPENAI_API_KEY", cfg.APIKey); key != "" {
		openaiCfg := cfg
		openaiCfg.APIKey = key
		p, err := NewOpenAIProvider(openaiCfg)
		if err == nil {
			return &AutoProvider{provider: p}, nil
		}
	}

	return nil, fmt.Errorf("no LLM API key found (set ANTHROPIC_API_KEY or OPENAI_API_KEY)")
}

func (a *AutoProvider) Name() string {
	return "auto:" + a.provider.Name()
}

func (a *AutoProvider) Parse(ctx context.Context, input string, commands []*engine.Command) (*ParseResult, error) {
	return a.provider.Parse(ctx, input, commands)
}

// getAPIKey returns the API key from env var or config.
func getAPIKey(envVar, configKey string) string {
	if configKey != "" {
		return configKey
	}
	return os.Getenv(envVar)
}

// buildSystemPrompt creates the system prompt for command parsing.
func buildSystemPrompt(commands []*engine.Command) string {
	prompt := `You are a CLI command parser. Your task is to interpret natural language input and determine which command the user wants to execute.

Available commands:
`
	for _, cmd := range commands {
		prompt += fmt.Sprintf("\n- %s: %s", cmd.Name, cmd.Description)
		if len(cmd.Examples) > 0 {
			prompt += "\n  Examples: "
			for i, ex := range cmd.Examples {
				if i > 0 {
					prompt += ", "
				}
				prompt += fmt.Sprintf("\"%s\"", ex)
			}
		}
	}

	prompt += `

Analyze the user's input and call the appropriate command tool with the correct parameters.
If the input is ambiguous or doesn't match any command, respond with an explanation.`

	return prompt
}
