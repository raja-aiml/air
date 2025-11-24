package containers

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

// StartServerInBackground starts the application server in a goroutine
// and signals via the ready channel when the server is healthy
func StartServerInBackground(ctx context.Context, cfg *Config, infra *Infrastructure, ready chan<- struct{}) error {
	// Kill any existing process on the configured port
	serverPort := cfg.ServerPort
	if !isPortAvailable(serverPort) {
		fmt.Printf("Port %s in use, killing existing process...\n", serverPort)
		killProcessOnPort(serverPort)
		time.Sleep(1 * time.Second)

		// If still not available, try other ports
		if !isPortAvailable(serverPort) {
			for port := 8080; port <= 8090; port++ {
				portStr := fmt.Sprintf("%d", port)
				if isPortAvailable(portStr) {
					serverPort = portStr
					fmt.Printf("Using port %s instead\n", serverPort)
					break
				}
			}
			if !isPortAvailable(serverPort) {
				return fmt.Errorf("no available port found in range 8080-8090")
			}
		}
	}

	// Set environment variables
	if cfg.OTELEnabled {
		// Verify OTEL endpoint is reachable before starting server (silent check)
		conn, err := net.DialTimeout("tcp", infra.OtelEndpoint, 5*time.Second)
		if err != nil {
			return fmt.Errorf("OTEL endpoint %s not reachable: %w", infra.OtelEndpoint, err)
		}
		conn.Close()

		os.Setenv("OTEL_ENABLED", "true")
		os.Setenv("OTEL_ENDPOINT", infra.OtelEndpoint)
		os.Setenv("OTEL_SERVICE_NAME", cfg.OTELServiceName)
		os.Setenv("OTEL_ENVIRONMENT", cfg.OTELEnvironment)
		os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")    // Disable TLS for local testing
		os.Setenv("OTEL_EXPORTER_OTLP_TRACES_SYNC", "true") // Use sync exporter for immediate trace delivery
	}
	os.Setenv("DATABASE_URL", infra.PostgresURL)
	os.Setenv("JWT_SECRET", cfg.JWTSecret)
	os.Setenv("JWT_ISS", cfg.JWTIssuer)
	os.Setenv("JWT_AUD", cfg.JWTAudience)
	os.Setenv("PORT", serverPort)

	// Set any extra environment variables
	for k, v := range cfg.ExtraEnv {
		os.Setenv(k, v)
	}

	// Update config with actual port used
	cfg.ServerPort = serverPort

	// Start server in goroutine
	serverLogFile, err := os.Create("logs/server-verify.log")
	if err != nil {
		return fmt.Errorf("create server log file: %w", err)
	}

	go func() {
		defer serverLogFile.Close()
		cmd := exec.CommandContext(ctx, cfg.ServerCommand[0], cfg.ServerCommand[1:]...)

		// Explicitly pass environment variables to subprocess
		cmd.Env = os.Environ() // Start with parent's environment
		cmd.Stdout = serverLogFile
		cmd.Stderr = serverLogFile

		if err := cmd.Run(); err != nil {
			// Context cancellation is expected during cleanup
			if ctx.Err() == nil {
				fmt.Printf("❌ Server exited with error: %v\n", err)
			}
		}
	}()

	// Wait for server to be ready
	healthURL := fmt.Sprintf("http://localhost:%s%s", serverPort, cfg.HealthEndpoint)
	if err := WaitForHTTP(ctx, healthURL, 15*time.Second); err != nil {
		return fmt.Errorf("server health check failed: %w", err)
	}

	// Give migrations extra time to complete (health check may respond before migrations finish)
	time.Sleep(3 * time.Second)

	// Signal that server is ready
	if ready != nil {
		close(ready)
	}

	return nil
}
