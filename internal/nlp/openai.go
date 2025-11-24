package nlp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/raja-aiml/air/internal/engine"
)

// OpenAIProvider implements Provider using OpenAI.
type OpenAIProvider struct {
	client openai.Client
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg ProviderConfig) (*OpenAIProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai API key required")
	}

	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
	)

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &OpenAIProvider{
		client: client,
		model:  model,
	}, nil
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) Parse(ctx context.Context, input string, commands []*engine.Command) (*ParseResult, error) {
	// Build function tools from commands
	tools := make([]openai.ChatCompletionToolParam, len(commands))
	for i, cmd := range commands {
		tools[i] = openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        cmd.Name,
				Description: openai.String(cmd.Description),
				Parameters:  shared.FunctionParameters(cmd.ParameterSchema()),
			},
		}
	}

	// Create chat completion with tools
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(buildSystemPrompt(commands)),
			openai.UserMessage(input),
		},
		Tools: tools,
	})
	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}

	// Extract tool call from response
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) > 0 {
			toolCall := choice.Message.ToolCalls[0]

			params := make(map[string]any)
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
				return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
			}

			return &ParseResult{
				Command:    toolCall.Function.Name,
				Parameters: params,
				Confidence: 1.0,
				Source:     "openai",
				RawInput:   input,
			}, nil
		}

		// No tool call - return text response as error
		if choice.Message.Content != "" {
			return nil, fmt.Errorf("could not parse command: %s", choice.Message.Content)
		}
	}

	return nil, fmt.Errorf("no valid response from OpenAI")
}
