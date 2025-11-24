package tests

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/raja-aiml/air/internal/testinfra/containers"
)

// VerifyOtelCollectorLogs checks OTEL collector logs for export errors
func VerifyOtelCollectorLogs(t TestingT, ctx context.Context, infra *containers.Infrastructure) error {
	// Get container logs (works with both testcontainers and Docker SDK)
	logs, err := infra.GetContainerLogs(ctx, "otel")
	if err != nil {
		return fmt.Errorf("failed to get OTEL logs: %w", err)
	}
	defer logs.Close()

	// Read all logs
	logBytes, err := io.ReadAll(logs)
	if err != nil {
		return fmt.Errorf("failed to read OTEL logs: %w", err)
	}

	logContent := string(logBytes)

	// Check for common errors (including Jaeger export errors)
	errorPatterns := []string{
		"connection refused",
		"dial tcp.*failed",
		"TLS handshake",
		"authentication handshake failed",
		"no such host",
		"connection error",
		"Exporting failed",
		"failed to export",
		"error sending spans",
	}

	foundErrors := []string{}
	for _, pattern := range errorPatterns {
		if strings.Contains(strings.ToLower(logContent), strings.ToLower(pattern)) {
			// Extract the line with the error
			lines := strings.Split(logContent, "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
					foundErrors = append(foundErrors, line)
					break
				}
			}
		}
	}

	if len(foundErrors) > 0 {
		t.Errorf("OTEL Collector has connection errors:")
		for _, err := range foundErrors {
			t.Errorf("  - %s", err)
		}
		return fmt.Errorf("OTEL collector has %d connection errors", len(foundErrors))
	}

	// Silent success - only report if there are errors
	return nil
}

// VerifyJaegerLogs checks Jaeger logs for OTLP receiver status
func VerifyJaegerLogs(t TestingT, ctx context.Context, infra *containers.Infrastructure) error {
	// Get container logs (works with both testcontainers and Docker SDK)
	logs, err := infra.GetContainerLogs(ctx, "jaeger")
	if err != nil {
		return fmt.Errorf("failed to get Jaeger logs: %w", err)
	}
	defer logs.Close()

	logBytes, err := io.ReadAll(logs)
	if err != nil {
		return fmt.Errorf("failed to read Jaeger logs: %w", err)
	}

	logContent := string(logBytes)

	// Check if OTLP receiver is enabled
	if !strings.Contains(logContent, "OTLP") && !strings.Contains(logContent, "otlp") {
		t.Errorf("Jaeger logs don't mention OTLP receiver - may not be enabled")
		return fmt.Errorf("Jaeger OTLP receiver not detected in logs")
	}

	// Silent success
	return nil
}
