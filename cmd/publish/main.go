package main

import (
	"fmt"
	"os"

	pkg "github.com/raja-aiml/air/pkg"
)

func main() {
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
		fmt.Fprintln(os.Stderr, "publish failed:", err)
		os.Exit(1)
	}

	fmt.Println("Publish completed successfully")
}
