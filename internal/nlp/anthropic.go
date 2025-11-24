package nlp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/raja-aiml/air/internal/engine"
)

// AnthropicProvider implements Provider using Claude.
type AnthropicProvider struct {
	client anthropic.Client
	model  string
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg ProviderConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key required")
	}

	client := anthropic.NewClient(
		option.WithAPIKey(cfg.APIKey),
	)

	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	return &AnthropicProvider{
		client: client,
		model:  model,
	}, nil
}

func (a *AnthropicProvider) Name() string {
	return "anthropic"
}

func (a *AnthropicProvider) Parse(ctx context.Context, input string, commands []*engine.Command) (*ParseResult, error) {
	// Build tools from commands
	tools := make([]anthropic.ToolUnionParam, len(commands))
	for i, cmd := range commands {
		schema := cmd.ParameterSchema()
		properties, _ := schema["properties"]
		required, _ := schema["required"].([]string)

		tools[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        cmd.Name,
				Description: anthropic.String(cmd.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: properties,
					Required:   required,
				},
			},
		}
	}

	// Create message with tool use
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: buildSystemPrompt(commands)},
		},
		Tools: tools,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(input),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	// Extract tool use from response
	for _, block := range resp.Content {
		if block.Type == "tool_use" {
			params := make(map[string]any)
			if err := json.Unmarshal(block.Input, &params); err != nil {
				return nil, fmt.Errorf("failed to parse tool input: %w", err)
			}

			return &ParseResult{
				Command:    block.Name,
				Parameters: params,
				Confidence: 1.0, // LLM-based parsing is considered high confidence
				Source:     "anthropic",
				RawInput:   input,
			}, nil
		}
	}

	// No tool use found - extract text response
	for _, block := range resp.Content {
		if block.Type == "text" {
			return nil, fmt.Errorf("could not parse command: %s", block.Text)
		}
	}

	return nil, fmt.Errorf("no valid response from Claude")
}
