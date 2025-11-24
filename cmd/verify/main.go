package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	pkg "github.com/raja-aiml/air/pkg"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func cleanupPreviousRuns() {
	ctx := context.Background()

	// Kill any process on port 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		// Port in use, kill the process
		exec.Command("sh", "-c", "lsof -ti:8080 | xargs kill -9 2>/dev/null").Run()
		time.Sleep(500 * time.Millisecond)
	} else {
		listener.Close()
	}

	// Use Docker SDK to cleanup containers
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return // Silently fail if Docker not available
	}
	defer cli.Close()

	// Stop and remove skill-flow containers
	listOpts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project=skill-flow"),
		),
	}

	containers, err := cli.ContainerList(ctx, listOpts)
	if err == nil {
		timeout := 2
		for _, c := range containers {
			cli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout})
			cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
		}
	}

	// Remove skill-flow networks
	networks, err := cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", "skill-flow")),
	})
	if err == nil {
		for _, net := range networks {
			cli.NetworkRemove(ctx, net.ID)
		}
	}

	time.Sleep(1 * time.Second)
}

var (
	timeout = flag.Duration("timeout", 120*time.Second, "Overall timeout for verification")
)

func main() {
	flag.Parse()

	// Check for subcommands
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "down":
			runDown()
			return
		case "status":
			runStatus()
			return
		default:
			fmt.Printf("Unknown command: %s\n", args[0])
			fmt.Println("Usage: verify [down|status]")
			os.Exit(1)
		}
	}

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  🔬 OBSERVABILITY VERIFICATION")
	fmt.Println(strings.Repeat("═", 60))

	// Clean up any lingering processes and containers
	cleanupPreviousRuns()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Create logger that implements TestingT for command-line output
	logger := pkg.NewManualTester(true)

	// Create default config - loads everything from docker-compose.yml
	cfg := pkg.DefaultTestConfig()
	cfg.ServiceName = cfg.OTELServiceName // Match OTEL service name for Jaeger queries

	// Phase 1: Start infrastructure using StartWithCompose (all config from docker-compose.yml)
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("▶ Infrastructure Startup")
	fmt.Println(strings.Repeat("─", 60))
	phaseStart := time.Now()

	infra, err := pkg.StartWithCompose(ctx, cfg)
	if err != nil {
		fmt.Printf("❌ Failed to start infrastructure: %v\n", err)
		os.Exit(1)
	}
	defer pkg.CleanupInfrastructure(infra)

	fmt.Printf("  ✓ Postgres, Jaeger, Prometheus, OTEL Collector (%v)\n", time.Since(phaseStart).Round(10*time.Millisecond))

	// Wait for services to be actually ready (not just running)
	if err := pkg.WaitForPostgres(ctx, infra.PostgresURL); err != nil {
		fmt.Printf("❌ PostgreSQL not ready: %v\n", err)
		os.Exit(1)
	}
	if err := pkg.WaitForJaeger(ctx, infra.JaegerURL); err != nil {
		fmt.Printf("❌ Jaeger not ready: %v\n", err)
		os.Exit(1)
	}
	if err := pkg.WaitForPrometheus(ctx, infra.PrometheusURL); err != nil {
		fmt.Printf("❌ Prometheus not ready: %v\n", err)
		os.Exit(1)
	}

	// Phase 2: Verify container health
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("▶ Health Checks")
	fmt.Println(strings.Repeat("─", 60))
	phaseStart = time.Now()
	report := pkg.NewReport(false) // Verbose mode
	if err := pkg.VerifyContainerHealth(ctx, infra, report); err != nil {
		fmt.Printf("❌ Container health checks failed: %v\n", err)
		fmt.Printf("   ℹ️ Check logs: docker logs skill-flow-<service>\n")
		os.Exit(1)
	}
	fmt.Printf("  (%v)\n", time.Since(phaseStart).Round(10*time.Millisecond))

	// Phase 3: Start server in goroutine (server runs its own migrations)
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("▶ Application Server")
	fmt.Println(strings.Repeat("─", 60))
	phaseStart = time.Now()
	serverCtx, cancelServer := context.WithCancel(ctx)
	defer cancelServer()

	serverReady := make(chan struct{})
	if err := pkg.StartServerInBackground(serverCtx, cfg, infra, serverReady); err != nil {
		fmt.Printf("❌ Failed to start server: %v\n", err)
		fmt.Printf("   ℹ️ Check: lsof -ti:8080 (port may be in use)\n")
		os.Exit(1)
	}

	// Wait for server ready with timeout
	fmt.Print("  ⏳ Starting server and running migrations...")
	select {
	case <-serverReady:
		fmt.Printf("\r  ✓ Server ready (port 8080, telemetry enabled) (%v)\n", time.Since(phaseStart).Round(10*time.Millisecond))
	case <-time.After(20 * time.Second):
		fmt.Println("\n❌ Server startup timeout")
		fmt.Printf("   ℹ️ Check database connection: %s\n", infra.PostgresURL)
		os.Exit(1)
	}

	// Phase 5: Run verification tests using shared test functions
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("▶ Observability Pipeline")
	fmt.Println(strings.Repeat("─", 60))
	phaseStart = time.Now()
	fmt.Println()
	fmt.Println("  Data Flow:")
	fmt.Println("  ┌─────────────┐")
	fmt.Println("  │   Server    │ WebSocket traffic (connect → question → answer)")
	fmt.Println("  │  (port 8080)│")
	fmt.Println("  └──────┬──────┘")
	fmt.Println("         │")
	fmt.Println("         │ OTLP/gRPC")
	fmt.Println("         ▼")
	fmt.Println("  ┌─────────────┐")
	fmt.Println("  │    OTEL     │ Receives traces & metrics")
	fmt.Println("  │  Collector  │")
	fmt.Println("  └──────┬──────┘")
	fmt.Println("         │")
	fmt.Println("    ┌────┴────┐")
	fmt.Println("    │         │")
	fmt.Println("    ▼         ▼")
	fmt.Println("┌────────┐ ┌──────────┐")
	fmt.Println("│ Jaeger │ │Prometheus│ Storage & visualization")
	fmt.Println("│ (traces)│(metrics) │")
	fmt.Println("└────────┘ └──────────┘")
	fmt.Println()

	fmt.Println("\n  Testing:")
	if err := pkg.VerifyTracesPropagation(logger, ctx, cfg, infra); err != nil {
		fmt.Printf("  ❌ Traces verification failed: %v\n", err)
		fmt.Printf("     ℹ️ Check Jaeger UI: %s\n", infra.JaegerURL)
		os.Exit(1)
	}

	if err := pkg.VerifyMetricsCollection(logger, ctx, cfg, infra); err != nil {
		fmt.Printf("  ❌ Metrics verification failed: %v\n", err)
		fmt.Printf("     ℹ️ Check Prometheus UI: %s\n", infra.PrometheusURL)
		os.Exit(1)
	}

	fmt.Printf("\n  ✓ All pipeline tests passed (%v)\n", time.Since(phaseStart).Round(10*time.Millisecond))

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  ✅ VERIFICATION COMPLETE")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()
	fmt.Println("  Summary:")
	fmt.Println("    ✓ Infrastructure health verified")
	fmt.Println("    ✓ Server running with telemetry")
	fmt.Println("    ✓ Traces flowing to Jaeger")
	fmt.Println("    ✓ Metrics collected in Prometheus")
	fmt.Println("    ✓ End-to-end observability confirmed")
	fmt.Println()
	fmt.Println("  Observability UIs:")
	fmt.Printf("    → Jaeger:     %s\n", infra.JaegerURL)
	fmt.Printf("    → Prometheus: %s\n", infra.PrometheusURL)
}

// runDown stops all verification infrastructure
func runDown() {
	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  🛑 STOPPING VERIFICATION INFRASTRUCTURE")
	fmt.Println(strings.Repeat("═", 60))

	ctx := context.Background()

	// Kill any process on port 8080
	fmt.Print("  ⏳ Stopping server...")
	exec.Command("sh", "-c", "lsof -ti:8080 | xargs kill -9 2>/dev/null").Run()
	fmt.Println("\r  ✓ Server stopped    ")

	// Use Docker SDK to cleanup containers
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("❌ Failed to connect to Docker: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	// Stop and remove skill-flow containers
	fmt.Print("  ⏳ Stopping containers...")
	listOpts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project=skill-flow"),
		),
	}

	containers, err := cli.ContainerList(ctx, listOpts)
	if err != nil {
		fmt.Printf("\n❌ Failed to list containers: %v\n", err)
		os.Exit(1)
	}

	if len(containers) == 0 {
		fmt.Println("\r  ℹ️  No containers running")
	} else {
		timeout := 5
		for _, c := range containers {
			cli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout})
			cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
		}
		fmt.Printf("\r  ✓ Stopped %d containers\n", len(containers))
	}

	// Remove skill-flow networks
	fmt.Print("  ⏳ Removing networks...")
	networks, err := cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project=skill-flow"),
		),
	})
	if err == nil && len(networks) > 0 {
		for _, net := range networks {
			cli.NetworkRemove(ctx, net.ID)
		}
		fmt.Printf("\r  ✓ Removed %d networks\n", len(networks))
	} else {
		fmt.Println("\r  ℹ️  No networks to remove")
	}

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  ✅ CLEANUP COMPLETE")
	fmt.Println(strings.Repeat("═", 60))
}

// runStatus shows the status of verification infrastructure
func runStatus() {
	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  📊 VERIFICATION INFRASTRUCTURE STATUS")
	fmt.Println(strings.Repeat("═", 60))

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("❌ Failed to connect to Docker: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	listOpts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project=skill-flow"),
		),
	}

	containers, err := cli.ContainerList(ctx, listOpts)
	if err != nil {
		fmt.Printf("❌ Failed to list containers: %v\n", err)
		os.Exit(1)
	}

	if len(containers) == 0 {
		fmt.Println("\n  ℹ️  No containers running")
		fmt.Println("\n  Run 'verify' to start infrastructure")
	} else {
		fmt.Println()
		for _, c := range containers {
			stateIcon := "✓"
			if c.State != "running" {
				stateIcon = "⚠️"
			}
			name := strings.TrimPrefix(c.Names[0], "/")
			fmt.Printf("  %s %s (%s)\n", stateIcon, name, c.State)
		}
	}

	// Check if server is running
	fmt.Println()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("  ✓ Server running on port 8080")
	} else {
		listener.Close()
		fmt.Println("  ⚠️  Server not running")
	}

	fmt.Println()
}
