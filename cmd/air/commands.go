package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
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
		// Inline the previous publish/main.go logic
		fmt.Println("🚀 Publishing air to GitHub...")
		fmt.Println()

		// Open the repository
		repo, err := git.PlainOpen(".")
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		// Create GitHub API client
		client, err := api.DefaultRESTClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}

		// Create repository on GitHub
		fmt.Println("📦 Creating repository 'air' on GitHub...")
		repoData := map[string]interface{}{
			"name":        "air",
			"description": "AI Runtime Infrastructure - Build production-ready AI agents and MCP servers in Go with batteries-included observability",
			"private":     false,
		}

		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(repoData)
		err = client.Post("user/repos", &buf, nil)
		if err != nil {
			fmt.Printf("⚠️  Repository might already exist: %v\n", err)
			fmt.Println("   Continuing with existing repository...")
		} else {
			fmt.Println("✅ Repository created!")
		}
		fmt.Println()

		// Add topics
		fmt.Println("🏷️  Adding repository topics...")
		topics := map[string]interface{}{
			"names": []string{
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
		}

		var topicsBuf bytes.Buffer
		json.NewEncoder(&topicsBuf).Encode(topics)
		err = client.Put("repos/raja-aiml/air/topics", &topicsBuf, nil)
		if err != nil {
			fmt.Printf("⚠️  Failed to add topics: %v\n", err)
		} else {
			fmt.Println("✅ Topics added!")
		}
		fmt.Println()

		// Push to GitHub
		fmt.Println("⬆️  Pushing code to GitHub...")
		err = repo.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/main:refs/heads/main")},
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			fmt.Printf("⚠️  Failed to push: %v\n", err)
			fmt.Println("   You may need to push manually: git push -u origin main")
		} else {
			fmt.Println("✅ Code pushed!")
		}
		fmt.Println()

		// Create tag
		fmt.Println("🏷️  Creating release tag v0.1.0...")
		head, err := repo.Head()
		if err != nil {
			return fmt.Errorf("failed to get HEAD: %w", err)
		}

		tagMessage := `Release v0.1.0 - Initial release of air

Features:
- Full observability stack (OpenTelemetry, Jaeger, Prometheus)
- PostgreSQL with pgvector for AI embeddings
- Testing infrastructure with Testcontainers
- Docker Compose integration
- CLI tools for infrastructure management
- Production-ready foundation for AI agents and MCP servers`

		_, err = repo.CreateTag("v0.1.0", head.Hash(), &git.CreateTagOptions{
			Tagger: &object.Signature{
				Name:  "Raja",
				Email: "raja@aiml.com",
				When:  time.Now(),
			},
			Message: tagMessage,
		})
		if err != nil {
			fmt.Printf("⚠️  Failed to create tag: %v\n", err)
			fmt.Println("   Tag might already exist or you may need to create it manually")
		} else {
			fmt.Println("✅ Tag created!")
		}

		// Push tag
		fmt.Println("⬆️  Pushing tag to GitHub...")
		err = repo.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/v0.1.0:refs/tags/v0.1.0")},
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			fmt.Printf("⚠️  Failed to push tag: %v\n", err)
			fmt.Println("   You may need to push manually: git push origin v0.1.0")
		} else {
			fmt.Println("✅ Tag pushed!")
		}
		fmt.Println()

		fmt.Println("✅ Publishing complete!")
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
