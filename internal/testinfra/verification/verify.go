package verification

import (
	"context"
	"fmt"

	"github.com/raja-aiml/air/internal/testinfra/containers"
)

// Run executes the full observability verification workflow
func Run(ctx context.Context, cfg *containers.Config, jsonOutput bool) error {
	report := containers.NewReport(jsonOutput)

	// Phase 1: Start Containers
	report.Phase("Starting Infrastructure")
	infra, err := containers.StartInfrastructure(ctx, cfg, report)
	if err != nil {
		report.Fail("Container startup failed: %v", err)
		return fmt.Errorf("container startup: %w", err)
	}
	defer containers.CleanupInfrastructure(infra)

	// Phase 2: Verify Container Health (before starting server)
	if err := containers.VerifyContainerHealth(ctx, infra, report); err != nil {
		return fmt.Errorf("container health check: %w", err)
	}

	// Phase 3: Start Application Server
	if err := containers.StartApplicationServer(ctx, cfg, infra, report); err != nil {
		return fmt.Errorf("server startup: %w", err)
	}

	// Phase 4: Generate Traffic
	report.Phase("Generating Traffic")
	correlationIDs, err := containers.GenerateTraffic(ctx, cfg, infra, report)
	if err != nil {
		report.Fail("Traffic generation failed: %v", err)
		return fmt.Errorf("traffic generation: %w", err)
	}

	// Phase 5: Verify Data Flow Through Pipeline
	report.Phase("Verifying Data Flow")

	report.Step("Checking traces in Jaeger...")
	if err := VerifyJaegerTraces(ctx, cfg, infra.JaegerURL, correlationIDs, report); err != nil {
		report.Fail("Trace verification failed: %v", err)
		return fmt.Errorf("trace verification: %w", err)
	}
	report.Info("✓ Server → OTEL Collector → Jaeger")

	report.Step("Checking metrics in Prometheus...")
	if err := VerifyPrometheusMetrics(ctx, infra.PrometheusURL, report); err != nil {
		report.Fail("Metrics verification failed: %v", err)
		return fmt.Errorf("metrics verification: %w", err)
	}
	report.Info("✓ Server → OTEL Collector → Prometheus")

	report.Step("Checking server metrics endpoint...")
	if err := VerifyMetricsEndpoint(ctx, cfg, report); err != nil {
		report.Fail("Metrics endpoint verification failed: %v", err)
		return fmt.Errorf("metrics endpoint: %w", err)
	}
	report.StepSuccess("Complete data flow verified")

	// Final Report
	report.Phase("Verification Complete")
	report.Success("✅ All checks passed!")
	report.Info("  • Containers: healthy")
	report.Info("  • Server: running")
	report.Info("  • Traces: propagating to Jaeger")
	report.Info("  • Metrics: propagating to Prometheus")
	report.Print()

	return nil
}
