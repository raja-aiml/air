package containers

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for infrastructure setup
type Config struct {
	// Project identification
	ProjectName string
	ServiceName string

	// Database configuration
	DBUser     string
	DBPassword string
	DBName     string

	// Server configuration
	ServerPort     string
	ServerCommand  []string // e.g., []string{"go", "run", "cmd/server/main.go"}
	HealthEndpoint string   // e.g., "/healthz"

	// JWT configuration
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string

	// WebSocket configuration
	WSEndpoint string // e.g., "/ws"

	// OTEL configuration
	OTELEnabled     bool
	OTELServiceName string
	OTELEnvironment string

	// Docker Compose configuration
	ComposeFilePath string // Path to docker-compose.yml
	OtelConfigPath  string // Path to otel-collector-config.yaml

	// File paths
	MigrationsDir string // Path to database migrations
	SeedsDir      string // Path to database seeds

	// Parsed from docker-compose.yml
	ContainerImages map[string]string // Service name -> Docker image
	NetworkName     string            // Docker network name

	// Additional environment variables for server
	ExtraEnv map[string]string
}

// DockerComposeFile represents docker-compose.yml structure
type DockerComposeFile struct {
	Services map[string]struct {
		ContainerName string            `yaml:"container_name"`
		Image         string            `yaml:"image"`
		Environment   map[string]string `yaml:"environment"`
		Ports         []string          `yaml:"ports"`
	} `yaml:"services"`
	Networks map[string]interface{} `yaml:"networks"`
}

// DefaultConfig returns a configuration loaded from /config files
func DefaultConfig() *Config {
	cfg := &Config{
		// File paths (relative from project root)
		ComposeFilePath: "config/docker/docker-compose.yml",
		OtelConfigPath:  "config/observability/otel-collector-config.yaml",
		MigrationsDir:   "config/database/migrations",
		SeedsDir:        "config/database/seeds",

		// Server defaults (not in docker-compose)
		ServerPort:      "8080",
		ServerCommand:   []string{"go", "run", "cmd/server/main.go"},
		HealthEndpoint:  "/healthz",
		WSEndpoint:      "/ws",
		OTELEnabled:     true,
		OTELServiceName: "skillflow-backend",
		OTELEnvironment: "test",
		ExtraEnv:        make(map[string]string),
		ContainerImages: make(map[string]string),
	}

	// Load configuration from docker-compose.yml
	if err := cfg.LoadFromDockerCompose(); err != nil {
		panic(fmt.Sprintf("FATAL: Cannot load docker-compose.yml: %v\nEnsure /config/docker/docker-compose.yml exists and is valid", err))
	}

	return cfg
}

// LoadFromDockerCompose parses docker-compose.yml and populates config
func (c *Config) LoadFromDockerCompose() error {
	absPath := c.ComposeFilePath
	if !filepath.IsAbs(absPath) {
		if wd, err := os.Getwd(); err == nil {
			absPath = filepath.Join(wd, c.ComposeFilePath)
		}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read docker-compose: %w", err)
	}

	var compose DockerComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return fmt.Errorf("parse docker-compose: %w", err)
	}

	// Extract PostgreSQL service configuration
	postgres, ok := compose.Services["postgres"]
	if !ok {
		return fmt.Errorf("docker-compose.yml missing 'postgres' service")
	}
	c.ContainerImages["postgres"] = postgres.Image
	c.DBUser = postgres.Environment["POSTGRES_USER"]
	c.DBPassword = postgres.Environment["POSTGRES_PASSWORD"]
	c.DBName = postgres.Environment["POSTGRES_DB"]

	if c.DBUser == "" || c.DBPassword == "" || c.DBName == "" {
		return fmt.Errorf("docker-compose.yml postgres service missing POSTGRES_* environment variables")
	}

	// Extract Jaeger configuration
	if jaeger, ok := compose.Services["jaeger"]; ok {
		c.ContainerImages["jaeger"] = jaeger.Image
	} else {
		return fmt.Errorf("docker-compose.yml missing 'jaeger' service")
	}

	// Extract Prometheus configuration
	if prometheus, ok := compose.Services["prometheus"]; ok {
		c.ContainerImages["prometheus"] = prometheus.Image
	} else {
		return fmt.Errorf("docker-compose.yml missing 'prometheus' service")
	}

	// Extract OTEL Collector configuration
	if otel, ok := compose.Services["otel-collector"]; ok {
		c.ContainerImages["otel-collector"] = otel.Image
	} else {
		return fmt.Errorf("docker-compose.yml missing 'otel-collector' service")
	}

	// Extract network name (use first defined network)
	if len(compose.Networks) == 0 {
		return fmt.Errorf("docker-compose.yml missing networks section")
	}
	for name := range compose.Networks {
		c.NetworkName = name
		break
	}

	// Derive project settings from DB name
	c.ProjectName = c.DBName
	c.ServiceName = "backend"
	c.JWTSecret = "test-secret-" + c.DBName
	c.JWTIssuer = "skill-flow"       // Must match server's hardcoded value
	c.JWTAudience = "skill-flow-app" // Must match server's hardcoded value

	return nil
}
