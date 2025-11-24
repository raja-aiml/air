package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	pkg "github.com/raja-aiml/air/pkg"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify observability stack",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		return pkg.VerifyObservability(ctx)
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Delegate publish workflow to pkg.PublishRepo
		opts := pkg.PublishOptions{
			RepoPath: ".",
			Repository: pkg.RepositoryConfig{
				Owner:       "raja-aiml",
				Name:        "air",
				Description: "AI Runtime Infrastructure - Build production-ready AI agents and MCP servers in Go with batteries-included observability",
				Private:     false,
				HasIssues:   true,
				HasWiki:     true,
				Topics: []string{
					"golang",
					"ai",
					"mcp",
					"model-context-protocol",
					"observability",
					"opentelemetry",
					"ai-agents",
					"tracing",
					"metrics",
					"postgresql",
					"pgvector",
				},
			},
			Release: pkg.ReleaseConfig{
				Tag: "v0.1.0",
				Message: `Release v0.1.0 - Initial release of air

Features:
- Full observability stack (OpenTelemetry, Jaeger, Prometheus)
- PostgreSQL with pgvector for AI embeddings
- Testing infrastructure with Testcontainers
- Docker Compose integration
- CLI tools for infrastructure management
- Production-ready foundation for AI agents and MCP servers`,
				AuthorName:  "Raja",
				AuthorEmail: "raja@aiml.com",
			},
			Remote: "origin",
			Branch: "main",
		}

		if err := pkg.PublishRepo(opts); err != nil {
			return fmt.Errorf("publish failed: %w", err)
		}

		return nil
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		mcpMode, _ := cmd.Flags().GetBool("mcp")
		if !mcpMode {
			return fmt.Errorf("use --mcp flag to start MCP server")
		}

		registry, err := initializeRegistry()
		if err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Starting air MCP server...")
		server := pkg.NewMCPServer(registry, pkg.DefaultMCPConfig())
		return server.ServeStdio(ctx)
	},
}

var nlpCmd = &cobra.Command{
	Use:   "nlp [query]",
	Short: "Natural language command processing",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		registry, err := initializeRegistry()
		if err != nil {
			return err
		}

		input := strings.Join(args, " ")

		parser, err := pkg.NewParser(registry, pkg.DefaultParserConfig())
		if err != nil {
			return fmt.Errorf("failed to initialize NLP parser: %w", err)
		}

		fmt.Printf("Parsing: %q\n", input)
		if parser.HasLLMProvider() {
			fmt.Printf("Using LLM provider: %s\n", parser.ProviderName())
		} else {
			fmt.Println("Using local embeddings only (no LLM API key found)")
		}

		result, err := parser.Parse(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to parse command: %w", err)
		}

		fmt.Printf("Matched command: %s (confidence: %.2f, source: %s)\n\n",
			result.Command, result.Confidence, result.Source)

		execResult, err := registry.Execute(ctx, result.Command, result.Parameters)
		if err != nil {
			return err
		}

		fmt.Println(execResult.Message)
		return nil
	},
}

var execCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Execute a command directly",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		registry, err := initializeRegistry()
		if err != nil {
			return err
		}

		// Convert args to command name and params
		var cmdName string
		var cmdArgs []string

		if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
			cmdName = args[0] + "." + args[1]
			cmdArgs = args[2:]
		} else if strings.Contains(args[0], ".") {
			cmdName = args[0]
			cmdArgs = args[1:]
		} else {
			return fmt.Errorf("unknown command: %s\nRun 'air help' for usage", args[0])
		}

		if _, ok := registry.Get(cmdName); !ok {
			return fmt.Errorf("unknown command: %s\nRun 'air commands' to see available commands", cmdName)
		}

		params := parseCommandFlags(cmdArgs)

		result, err := registry.Execute(ctx, cmdName, params)
		if err != nil {
			return err
		}

		fmt.Println(result.Message)
		return nil
	},
}

func init() {
	serveCmd.Flags().Bool("mcp", false, "Run as MCP server (stdio transport)")
}
