package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func main() {
	fmt.Println("🚀 Publishing air to GitHub...")
	fmt.Println()

	// Open the repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		fmt.Printf("❌ Failed to open repository: %v\n", err)
		os.Exit(1)
	}

	// Create GitHub API client
	client, err := api.DefaultRESTClient()
	if err != nil {
		fmt.Printf("❌ Failed to create GitHub client: %v\n", err)
		fmt.Println("   Please authenticate with GitHub CLI:")
		fmt.Println("   gh auth login")
		os.Exit(1)
	}

	// Create repository on GitHub
	fmt.Println("📦 Creating repository 'air' on GitHub...")
	repoData := map[string]interface{}{
		"name":          "air",
		"description":   "AI Runtime Infrastructure - Build production-ready AI agents and MCP servers in Go with batteries-included observability",
		"private":       false,
		"has_issues":    true,
		"has_wiki":      true,
		"has_downloads": true,
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
		fmt.Printf("❌ Failed to get HEAD: %v\n", err)
		os.Exit(1)
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
	fmt.Println()
	fmt.Println("📍 Repository URL: https://github.com/raja-aiml/air")
	fmt.Println("📦 Package will be available at: https://pkg.go.dev/github.com/raja-aiml/air")
	fmt.Println("🚀 Release: https://github.com/raja-aiml/air/releases/tag/v0.1.0")
	fmt.Println()
	fmt.Println("⏳ GitHub Actions will now:")
	fmt.Println("   - Run tests and linting")
	fmt.Println("   - Build binaries for all platforms")
	fmt.Println("   - Create GitHub Release with downloads")
	fmt.Println()
	fmt.Println("📦 Users can install with:")
	fmt.Println("   go get github.com/raja-aiml/air@latest")
	fmt.Println()
}
