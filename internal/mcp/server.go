// Package mcp provides an MCP (Model Context Protocol) server for air.
// Uses the official MCP Go SDK from github.com/modelcontextprotocol/go-sdk
package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/raja-aiml/air/internal/engine"
)

// Server wraps the MCP server and exposes commands as tools.
type Server struct {
	registry  *engine.Registry
	mcpServer *mcp.Server
}

// Config holds MCP server configuration.
type Config struct {
	Name    string
	Version string
}

// DefaultConfig returns default MCP server configuration.
func DefaultConfig() Config {
	return Config{
		Name:    "air",
		Version: "1.0.0",
	}
}

// NewServer creates a new MCP server from a command registry.
func NewServer(registry *engine.Registry, cfg Config) *Server {
	// Create MCP server with name and version
	mcpServer := mcp.NewServer(cfg.Name, cfg.Version, nil)

	s := &Server{
		registry:  registry,
		mcpServer: mcpServer,
	}

	// Register all commands as tools
	s.registerTools()

	return s
}

// ToolInput represents the input for a tool call.
type ToolInput struct {
	Parameters map[string]any `json:"parameters"`
}

// ToolOutput represents the output from a tool call.
type ToolOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// registerTools converts all registry commands to MCP tools.
func (s *Server) registerTools() {
	for _, cmd := range s.registry.All() {
		s.registerTool(cmd)
	}
}

// registerTool registers a single command as an MCP tool.
func (s *Server) registerTool(cmd *engine.Command) {
	// Capture cmd in closure
	command := cmd

	// Create handler function with properly typed parameters
	handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		// Extract parameters from request arguments
		args := params.Arguments
		if args == nil {
			args = make(map[string]any)
		}

		// Execute the command
		result, err := s.registry.Execute(ctx, command.Name, args)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Error: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Format output
		text := result.Message
		if result.Data != nil {
			text = fmt.Sprintf("%s\n\nData: %+v", result.Message, result.Data)
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: text,
				},
			},
			IsError: !result.Success,
		}, nil
	}

	// Register the tool with the server using NewServerTool
	serverTool := mcp.NewServerTool[map[string]any, any](command.Name, command.Description, handler)
	s.mcpServer.AddTools(serverTool)
}

// ServeStdio starts the MCP server using stdio transport.
func (s *Server) ServeStdio(ctx context.Context) error {
	transport := &mcp.StdioTransport{}
	return s.mcpServer.Run(ctx, transport)
}

// GetMCPServer returns the underlying MCP server for custom configuration.
func (s *Server) GetMCPServer() *mcp.Server {
	return s.mcpServer
}
