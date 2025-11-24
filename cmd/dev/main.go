package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pkg "github.com/raja-aiml/air/pkg"
)

func main() {
	// Define flags
	detach := flag.Bool("d", false, "Run in detached mode (background)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  up       Start services\n")
		fmt.Fprintf(os.Stderr, "  down     Stop and remove services\n")
		fmt.Fprintf(os.Stderr, "  status   Show service status\n")
		fmt.Fprintf(os.Stderr, "  logs     Show logs for a service\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	// Parse command
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	command := os.Args[1]
	flag.CommandLine.Parse(os.Args[2:])

	// Execute command
	switch command {
	case "up":
		runUp(*detach)
	case "down":
		runDown()
	case "status":
		runStatus()
	case "logs":
		if flag.NArg() < 1 {
			fmt.Println("❌ Usage: dev logs <service-name>")
			os.Exit(1)
		}
		runLogs(flag.Arg(0))
	default:
		fmt.Printf("❌ Unknown command: %s\n\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func runUp(detach bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Visual header
	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  🚀 STARTING DEVELOPMENT SERVICES")
	fmt.Println(strings.Repeat("═", 60))

	// Create compose service
	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	// Start services
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("▶ Starting Services")
	fmt.Println(strings.Repeat("─", 60))

	startTime := time.Now()
	if err := svc.Start(ctx); err != nil {
		fmt.Printf("❌ Failed to start: %v\n", err)
		os.Exit(1)
	}

	// Wait for health
	fmt.Print("  ⏳ Waiting for services to be healthy...")
	if err := svc.WaitForHealthy(ctx, 60*time.Second); err != nil {
		fmt.Printf("\n❌ Health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\r  ✓ All services healthy (%v)\n", time.Since(startTime).Round(10*time.Millisecond))

	// Display status
	status, err := svc.Status(ctx)
	if err == nil && status != nil {
		fmt.Println("\n" + strings.Repeat("─", 60))
		fmt.Println("▶ Service Status")
		fmt.Println(strings.Repeat("─", 60))

		for name, info := range status.Services {
			fmt.Printf("  ✓ %s\n", name)
			if info.ContainerID != "" {
				fmt.Printf("    Container: %s\n", info.ContainerID)
			}
			if len(info.Ports) > 0 {
				fmt.Printf("    Ports:     %s\n", strings.Join(info.Ports, ", "))
			}
			if info.HealthURL != "" {
				fmt.Printf("    URL:       %s\n", info.HealthURL)
			}
			fmt.Println()
		}
	}

	// Summary
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println("  ✅ READY")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()

	if status != nil {
		fmt.Println("  Quick Links:")
		for _, info := range status.Services {
			if info.HealthURL != "" {
				fmt.Printf("    → %s: %s\n", info.Name, info.HealthURL)
			}
		}
		fmt.Println()
	}

	if detach {
		fmt.Println("  Running in background. Use 'dev down' to stop.")
		fmt.Println()
		return
	}

	fmt.Println("  Press Ctrl+C to exit (services will keep running).")
	fmt.Println("  Use 'dev down' to stop all services.")
	fmt.Println()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n  ℹ️  Services still running. Use 'dev down' to stop.")
}

func runDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  🛑 STOPPING SERVICES")
	fmt.Println(strings.Repeat("═", 60))

	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	// Check status
	status, err := svc.Status(ctx)
	if err == nil && status != nil && len(status.Services) == 0 {
		fmt.Println("\n  ℹ️  No services running")
		return
	}

	// Stop
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("▶ Stopping")
	fmt.Println(strings.Repeat("─", 60))

	startTime := time.Now()
	fmt.Print("  ⏳ Stopping containers...")
	if err := svc.Stop(ctx); err != nil {
		fmt.Printf("\n❌ Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\r  ✓ Stopped (%v)\n", time.Since(startTime).Round(10*time.Millisecond))

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  ✅ CLEANUP COMPLETE")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()
}

func runStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	status, err := svc.Status(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get status: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  📊 SERVICE STATUS")
	fmt.Println(strings.Repeat("═", 60))

	if len(status.Services) == 0 {
		fmt.Println("\n  ℹ️  No services running")
		fmt.Println()
		return
	}

	fmt.Println()
	for name, info := range status.Services {
		stateIcon := "✓"
		if info.State != "running" {
			stateIcon = "⚠️"
		}
		fmt.Printf("  %s %s (%s)\n", stateIcon, name, info.State)
		if info.ContainerID != "" {
			fmt.Printf("    Container: %s\n", info.ContainerID)
		}
		if len(info.Ports) > 0 {
			fmt.Printf("    Ports:     %s\n", strings.Join(info.Ports, ", "))
		}
		if info.HealthURL != "" {
			fmt.Printf("    URL:       %s\n", info.HealthURL)
		}
		fmt.Println()
	}
}

func runLogs(serviceName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc, err := pkg.NewComposeService(pkg.ComposeConfig{
		ComposeFilePath: "config/docker/docker-compose.yml",
		ProjectName:     "skillflow",
		Env:             make(map[string]string),
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	logs, err := svc.Logs(ctx, serviceName)
	if err != nil {
		fmt.Printf("❌ Failed to get logs: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(logs)
}
