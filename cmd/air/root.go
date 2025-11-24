package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pkg "github.com/raja-aiml/air/pkg"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "air",
		Short: "AI Runtime CLI",
		Long:  "air — AI Runtime CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			// default: show help
			return cmd.Help()
		},
	}

	// persistent flags (can be used by subcommands)
	flagDatabaseURL string
	flagComposeFile string
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagDatabaseURL, "database-url", os.Getenv("DATABASE_URL"), "Postgres connection URL")
	rootCmd.PersistentFlags().StringVar(&flagComposeFile, "compose-file", os.Getenv("AIR_COMPOSE_FILE"), "Path to docker-compose.yml")

	// add subcommands
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(publishCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(nlpCmd)
	rootCmd.AddCommand(execCmd)
}

// initializeRegistry creates the command registry with all commands.
func initializeRegistry() (*pkg.Registry, error) {
	registry := pkg.NewRegistry()

	// Get configuration from flags or environment
	databaseURL := flagDatabaseURL
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/skillflow?sslmode=disable"
	}

	composeFile := flagComposeFile
	if composeFile == "" {
		// Try to find compose file relative to working directory
		candidates := []string{
			"config/docker/compose-template.yml",
			"ai-runtime/config/docker/compose-template.yml",
			"../config/docker/compose-template.yml",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				composeFile = c
				break
			}
		}
	}

	// Initialize compose service if config exists
	var composeSvc *pkg.ComposeService
	if composeFile != "" {
		absPath, _ := filepath.Abs(composeFile)
		cfg := pkg.ComposeConfig{
			ComposeFilePath: absPath,
			ProjectName:     "air",
		}
		svc, err := pkg.NewComposeService(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not initialize Docker Compose: %v\n", err)
		} else {
			composeSvc = svc
		}
	}

	// Register all command groups via pkg re-exports
	if composeSvc != nil {
		pkg.NewInfraCommands(composeSvc).Register(registry)
	}
	pkg.NewDBCommands(databaseURL).Register(registry)
	pkg.NewObsCommands().Register(registry)
	pkg.NewLintCommands().Register(registry)

	return registry, nil
}

// helper: parse flags for direct command execution
func parseCommandFlags(args []string) map[string]any {
	params := make(map[string]any)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}

		key := strings.TrimLeft(arg, "-")
		if idx := strings.Index(key, "="); idx > 0 {
			params[key[:idx]] = key[idx+1:]
			continue
		}

		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			value := args[i+1]
			if d, err := time.ParseDuration(value); err == nil {
				params[key] = d
			} else {
				params[key] = value
			}
			i++
		} else {
			params[key] = true
		}
	}

	return params
}

// helper: print registry commands
func printCommands(registry *pkg.Registry) {
	fmt.Println("Available Commands:")
	fmt.Println()

	cmds := registry.All()
	groups := make(map[string][]*pkg.Command)

	for _, cmd := range cmds {
		parts := strings.Split(cmd.Name, ".")
		group := parts[0]
		groups[group] = append(groups[group], cmd)
	}

	for group, cmds := range groups {
		fmt.Printf("  %s:\n", group)
		for _, cmd := range cmds {
			fmt.Printf("    %-20s %s\n", cmd.Name, cmd.Description)
		}
		fmt.Println()
	}
}
