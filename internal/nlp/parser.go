package nlp

import (
	"context"
	"fmt"

	"github.com/raja-aiml/air/internal/engine"
)

// Parser provides hybrid NLP parsing with local embeddings and LLM fallback.
type Parser struct {
	embeddings *EmbeddingMatcher
	provider   Provider
	registry   *engine.Registry
	threshold  float64
}

// ParserConfig holds configuration for the NLP parser.
type ParserConfig struct {
	Provider            ProviderConfig
	ConfidenceThreshold float64 // Minimum confidence for local matching (default: 0.7)
}

// DefaultParserConfig returns default parser configuration.
func DefaultParserConfig() ParserConfig {
	return ParserConfig{
		Provider:            DefaultConfig(),
		ConfidenceThreshold: 0.7,
	}
}

// NewParser creates a new hybrid NLP parser.
func NewParser(registry *engine.Registry, cfg ParserConfig) (*Parser, error) {
	if cfg.ConfidenceThreshold == 0 {
		cfg.ConfidenceThreshold = 0.7
	}

	// Initialize embeddings matcher
	embeddings := NewEmbeddingMatcher(registry.All())

	// Initialize LLM provider (may fail if no API key)
	provider, err := NewProvider(cfg.Provider)
	if err != nil {
		// LLM provider is optional - we can still use embeddings
		provider = nil
	}

	return &Parser{
		embeddings: embeddings,
		provider:   provider,
		registry:   registry,
		threshold:  cfg.ConfidenceThreshold,
	}, nil
}

// Parse interprets natural language input and returns the matching command.
func (p *Parser) Parse(ctx context.Context, input string) (*ParseResult, error) {
	// Step 1: Try local embeddings first (fast, free, offline)
	result, err := p.embeddings.Match(input)
	if err == nil && result.Confidence >= p.threshold {
		return result, nil
	}

	// Step 2: Fall back to LLM for ambiguous cases
	if p.provider != nil {
		llmResult, err := p.provider.Parse(ctx, input, p.registry.All())
		if err == nil {
			return llmResult, nil
		}
		// Log LLM error but don't fail - return best embedding match
		fmt.Printf("LLM fallback failed: %v\n", err)
	}

	// Step 3: Return best embedding match even if below threshold
	if result != nil && result.Command != "" {
		return result, nil
	}

	return nil, fmt.Errorf("could not parse command from input: %s", input)
}

// ParseWithoutLLM forces local-only parsing (useful for offline mode).
func (p *Parser) ParseWithoutLLM(input string) (*ParseResult, error) {
	return p.embeddings.Match(input)
}

// HasLLMProvider returns true if an LLM provider is available.
func (p *Parser) HasLLMProvider() bool {
	return p.provider != nil
}

// ProviderName returns the name of the active LLM provider.
func (p *Parser) ProviderName() string {
	if p.provider == nil {
		return "none"
	}
	return p.provider.Name()
}
