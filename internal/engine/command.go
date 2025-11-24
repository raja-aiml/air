// Package engine provides the core command execution framework for air CLI.
package engine

import (
	"context"
	"time"
)

// Command represents an executable command in the air CLI.
type Command struct {
	// Name is the unique identifier for this command (e.g., "infra.start")
	Name string

	// Description is a human-readable description for help and LLM context
	Description string

	// Examples are natural language examples for NLP training
	Examples []string

	// Parameters defines the inputs this command accepts
	Parameters []Parameter

	// Execute is the function that performs the command
	Execute func(ctx context.Context, params map[string]any) (Result, error)
}

// Parameter defines an input parameter for a command.
type Parameter struct {
	Name        string
	Type        string // "string", "bool", "int", "duration", "[]string"
	Required    bool
	Default     any
	Description string
}

// Result represents the outcome of a command execution.
type Result struct {
	Success  bool
	Message  string
	Data     any
	Duration time.Duration
}

// NewResult creates a successful result with a message.
func NewResult(message string) Result {
	return Result{
		Success: true,
		Message: message,
	}
}

// NewResultWithData creates a successful result with data.
func NewResultWithData(message string, data any) Result {
	return Result{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// ErrorResult creates a failed result from an error.
func ErrorResult(err error) Result {
	return Result{
		Success: false,
		Message: err.Error(),
	}
}

// ParameterSchema returns a JSON schema representation for LLM tool use.
func (c *Command) ParameterSchema() map[string]any {
	properties := make(map[string]any)
	required := []string{}

	for _, p := range c.Parameters {
		prop := map[string]any{
			"description": p.Description,
		}

		switch p.Type {
		case "string":
			prop["type"] = "string"
		case "bool":
			prop["type"] = "boolean"
		case "int":
			prop["type"] = "integer"
		case "duration":
			prop["type"] = "string"
			prop["format"] = "duration"
		case "[]string":
			prop["type"] = "array"
			prop["items"] = map[string]any{"type": "string"}
		default:
			prop["type"] = "string"
		}

		if p.Default != nil {
			prop["default"] = p.Default
		}

		properties[p.Name] = prop

		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}
