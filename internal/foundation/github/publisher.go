package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// RepositoryConfig contains configuration for creating a GitHub repository
type RepositoryConfig struct {
	Owner       string
	Name        string
	Description string
	Private     bool
	HasIssues   bool
	HasWiki     bool
	Topics      []string
}

// ReleaseConfig contains configuration for creating a release
type ReleaseConfig struct {
	Tag         string
	Message     string
	AuthorName  string
	AuthorEmail string
}

// Publisher handles GitHub repository publishing operations
type Publisher struct {
	client *api.RESTClient
	repo   *git.Repository
}

// NewPublisher creates a new GitHub publisher
func NewPublisher(repoPath string) (*Publisher, error) {
	// Open git repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Create GitHub API client
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client (authenticate with: gh auth login): %w", err)
	}

	return &Publisher{
		client: client,
		repo:   repo,
	}, nil
}

// CreateRepository creates a new GitHub repository
func (p *Publisher) CreateRepository(cfg RepositoryConfig) error {
	repoData := map[string]interface{}{
		"name":          cfg.Name,
		"description":   cfg.Description,
		"private":       cfg.Private,
		"has_issues":    cfg.HasIssues,
		"has_wiki":      cfg.HasWiki,
		"has_downloads": true,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(repoData); err != nil {
		return fmt.Errorf("failed to encode repository data: %w", err)
	}

	if err := p.client.Post("user/repos", &buf, nil); err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	return nil
}

// AddTopics adds topics to a GitHub repository
func (p *Publisher) AddTopics(owner, repo string, topics []string) error {
	topicsData := map[string]interface{}{
		"names": topics,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(topicsData); err != nil {
		return fmt.Errorf("failed to encode topics: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/topics", owner, repo)
	if err := p.client.Put(endpoint, &buf, nil); err != nil {
		return fmt.Errorf("failed to add topics: %w", err)
	}

	return nil
}

// PushCode pushes code to GitHub
func (p *Publisher) PushCode(remote, branch string) error {
	refSpec := config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branch, branch))
	err := p.repo.Push(&git.PushOptions{
		RemoteName: remote,
		RefSpecs:   []config.RefSpec{refSpec},
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push code: %w", err)
	}

	return nil
}

// CreateTag creates a git tag
func (p *Publisher) CreateTag(cfg ReleaseConfig) error {
	head, err := p.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	_, err = p.repo.CreateTag(cfg.Tag, head.Hash(), &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  cfg.AuthorName,
			Email: cfg.AuthorEmail,
			When:  time.Now(),
		},
		Message: cfg.Message,
	})

	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return nil
}

// PushTag pushes a tag to GitHub
func (p *Publisher) PushTag(remote, tag string) error {
	refSpec := config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", tag, tag))
	err := p.repo.Push(&git.PushOptions{
		RemoteName: remote,
		RefSpecs:   []config.RefSpec{refSpec},
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push tag: %w", err)
	}

	return nil
}

// PublishOptions contains all options for a complete publish workflow
type PublishOptions struct {
	RepoPath   string
	Repository RepositoryConfig
	Release    ReleaseConfig
	Remote     string
	Branch     string
}

// Publish executes a complete publish workflow
func Publish(opts PublishOptions) error {
	publisher, err := NewPublisher(opts.RepoPath)
	if err != nil {
		return err
	}

	// Create repository
	if err := publisher.CreateRepository(opts.Repository); err != nil {
		// Repository might already exist, continue
		fmt.Printf("Warning: %v (continuing...)\n", err)
	}

	// Add topics
	if len(opts.Repository.Topics) > 0 {
		if err := publisher.AddTopics(opts.Repository.Owner, opts.Repository.Name, opts.Repository.Topics); err != nil {
			return fmt.Errorf("failed to add topics: %w", err)
		}
	}

	// Push code
	if err := publisher.PushCode(opts.Remote, opts.Branch); err != nil {
		return fmt.Errorf("failed to push code: %w", err)
	}

	// Create and push tag
	if opts.Release.Tag != "" {
		if err := publisher.CreateTag(opts.Release); err != nil {
			fmt.Printf("Warning: %v (continuing...)\n", err)
		}

		if err := publisher.PushTag(opts.Remote, opts.Release.Tag); err != nil {
			return fmt.Errorf("failed to push tag: %w", err)
		}
	}

	return nil
}
