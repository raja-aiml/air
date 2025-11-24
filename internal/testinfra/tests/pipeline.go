package tests

import (
	"github.com/raja-aiml/air/internal/testinfra/containers"
	"github.com/raja-aiml/air/internal/testinfra/verification"
	"context"
	"fmt"
)

// VerifyTracesPropagation generates traffic and verifies traces reach Jaeger
func VerifyTracesPropagation(t TestingT, ctx context.Context, cfg *containers.Config, infra *containers.Infrastructure) error {
	report := containers.NewReport(false) // Use verbose mode, not JSON

	correlationIDs, err := containers.GenerateTraffic(ctx, cfg, infra, report)
	if err != nil {
		return fmt.Errorf("generate traffic: %w", err)
	}

	// Check OTEL collector logs before querying Jaeger (silent unless errors)
	// GetContainerLogs handles both testcontainers and Docker SDK paths
	if err := VerifyOtelCollectorLogs(t, ctx, infra); err != nil {
		t.Logf("⚠️  OTEL Collector has issues: %v", err)
		// Continue anyway to check Jaeger
	}

	// Check Jaeger logs (silent unless errors)
	// GetContainerLogs handles both testcontainers and Docker SDK paths
	if err := VerifyJaegerLogs(t, ctx, infra); err != nil {
		t.Logf("⚠️  Jaeger has issues: %v", err)
	}

	if err := verification.VerifyJaegerTraces(ctx, cfg, infra.JaegerURL, correlationIDs, report); err != nil {
		return fmt.Errorf("verify jaeger traces: %w", err)
	}

	return nil
}

// VerifyMetricsCollection verifies metrics are collected in Prometheus, OTEL collector, and server /metrics endpoint
func VerifyMetricsCollection(t TestingT, ctx context.Context, cfg *containers.Config, infra *containers.Infrastructure) error {
	report := containers.NewReport(false) // Use verbose mode, not JSON

	if err := verification.VerifyPrometheusMetrics(ctx, infra.PrometheusURL, report); err != nil {
		return fmt.Errorf("verify prometheus: %w", err)
	}

	if err := verification.VerifyOTELMetricsEndpoint(ctx, infra.OtelMetricsURL, report); err != nil {
		return fmt.Errorf("verify OTEL metrics endpoint: %w", err)
	}

	if err := verification.VerifyMetricsEndpoint(ctx, cfg, report); err != nil {
		return fmt.Errorf("verify metrics endpoint: %w", err)
	}

	return nil
}
