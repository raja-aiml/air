package testinfra

import (
	"context"
	"testing"
	"time"

	"github.com/raja-aiml/air/internal/testinfra/containers"
	"github.com/raja-aiml/air/internal/testinfra/tests"
)

func TestObservabilityPipeline(t *testing.T) {
	ctx := context.Background()

	// Phase 1: Start infrastructure
	t.Log("🔵 Starting Infrastructure")
	cfg := containers.DefaultConfig()
	infra, err := containers.StartWithCompose(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to start infrastructure: %v", err)
	}
	defer containers.CleanupInfrastructure(infra)

	// Phase 2: Verify container health
	t.Log("🔵 Verifying Container Health")
	report := containers.NewReport(false) // Verbose mode
	if err := containers.VerifyContainerHealth(ctx, infra, report); err != nil {
		t.Fatalf("Container health checks failed: %v", err)
	}

	// Phase 3: Start server in goroutine (server runs its own migrations)
	t.Log("🔵 Starting Application Server")
	serverCtx, cancelServer := context.WithCancel(ctx)
	defer cancelServer()

	serverReady := make(chan struct{})
	if err := containers.StartServerInBackground(serverCtx, cfg, infra, serverReady); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Wait for server ready with timeout
	select {
	case <-serverReady:
		t.Log("✅ Server ready")
	case <-time.After(20 * time.Second):
		t.Fatal("Server startup timeout")
	}

	// Phase 5: Run verification tests
	t.Run("TracesPropagation", func(t *testing.T) {
		if err := tests.VerifyTracesPropagation(t, ctx, cfg, infra); err != nil {
			t.Fatalf("Traces verification failed: %v", err)
		}
	})

	t.Run("MetricsCollection", func(t *testing.T) {
		if err := tests.VerifyMetricsCollection(t, ctx, cfg, infra); err != nil {
			t.Fatalf("Metrics verification failed: %v", err)
		}
	})

	t.Log("\n✅ All observability pipeline tests passed!")
}
