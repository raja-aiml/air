package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	pkg "github.com/raja-aiml/air/pkg"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development helpers (compose)",
	Long:  "Start/stop local development services via Docker Compose",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		action := args[0]
		switch action {
		case "up":
			return devUp()
		case "down":
			return devDown()
		case "status":
			return devStatus()
		case "logs":
			if len(args) < 2 {
				return fmt.Errorf("usage: air dev logs <service>")
			}
			return devLogs(args[1])
		default:
			return fmt.Errorf("unknown dev action: %s", action)
		}
	},
}

func devUp() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer svc.Close()

	start := time.Now()
	if err := svc.Start(ctx); err != nil {
		return err
	}

	if err := svc.WaitForHealthy(ctx, 60*time.Second); err != nil {
		return err
	}

	fmt.Printf("Services healthy (%v)\n", time.Since(start))
	return nil
}

func devDown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer svc.Close()

	if err := svc.Stop(ctx); err != nil {
		return err
	}

	fmt.Println("Services stopped")
	return nil
}

func devStatus() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer svc.Close()

	status, err := svc.Status(ctx)
	if err != nil {
		return err
	}

	for name, info := range status.Services {
		fmt.Printf("%s: %s\n", name, info.State)
	}
	return nil
}

func devLogs(service string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer svc.Close()

	logs, err := svc.Logs(ctx, service)
	if err != nil {
		return err
	}

	fmt.Println(strings.TrimSpace(logs))
	return nil
}
