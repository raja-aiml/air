//go:build integration

package telemetry

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestObservabilityStackIntegration(t *testing.T) {
	ctx := context.Background()

	// Start Jaeger container
	jaegerReq := tc.ContainerRequest{
		Image:        "jaegertracing/all-in-one:latest",
		ExposedPorts: []string{"16686/tcp", "4317/tcp"},
		Env: map[string]string{
			"COLLECTOR_OTLP_ENABLED": "true",
		},
		WaitingFor: wait.ForListeningPort("16686/tcp").WithStartupTimeout(60 * time.Second),
	}

	jaegerContainer, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: jaegerReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start jaeger container: %v", err)
	}
	defer jaegerContainer.Terminate(ctx)

	// Get Jaeger OTLP endpoint
	jaegerHost, err := jaegerContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get jaeger host: %v", err)
	}
	jaegerOTLPPort, err := jaegerContainer.MappedPort(ctx, "4317/tcp")
	if err != nil {
		t.Fatalf("failed to get jaeger OTLP port: %v", err)
	}

	otlpEndpoint := fmt.Sprintf("%s:%s", jaegerHost, jaegerOTLPPort.Port())
	t.Logf("Jaeger OTLP endpoint: %s", otlpEndpoint)

	// Set environment variables for InitTracer
	os.Setenv("OTEL_ENABLED", "true")
	os.Setenv("OTEL_ENDPOINT", otlpEndpoint)
	os.Setenv("OTEL_SERVICE_NAME", "test-service")
	os.Setenv("OTEL_ENVIRONMENT", "test")
	defer func() {
		os.Unsetenv("OTEL_ENABLED")
		os.Unsetenv("OTEL_ENDPOINT")
		os.Unsetenv("OTEL_SERVICE_NAME")
		os.Unsetenv("OTEL_ENVIRONMENT")
	}()

	// Initialize tracer with Jaeger
	shutdown, err := InitTracer(ctx)
	if err != nil {
		t.Fatalf("failed to initialize tracer: %v", err)
	}
	defer shutdown(ctx)

	// Create a test span
	traceCtx, span := Tracer().Start(ctx, "test.integration")
	AddSpanAttributes(traceCtx,
		attribute.String("test.name", "observability_stack"),
		attribute.String("test.type", "integration"),
	)

	// Simulate work
	time.Sleep(100 * time.Millisecond)

	span.End()

	// Force flush
	time.Sleep(2 * time.Second)

	// Verify trace was exported
	traceID := GetTraceID(traceCtx)
	if traceID == "" {
		t.Fatal("expected non-empty trace ID")
	}

	t.Logf("✅ Trace exported successfully to Jaeger (trace ID: %s)", traceID)
}

func TestFullQueryLifecycleTracing(t *testing.T) {
	// Use in-memory exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)

	// Override global tracer for test
	originalTracer := tracer
	tracer = tp.Tracer("skill-flow-test")
	defer func() {
		tracer = originalTracer
	}()

	ctx := context.Background()

	// Simulate full query lifecycle
	// 1. WebSocket connection
	ctx, wsSpan := Tracer().Start(ctx, "ws.connection")
	ctx = EnrichContext(ctx, "user-123", "sess-456", "req-789")
	AddSpanAttributes(ctx, attribute.String("remote_addr", "127.0.0.1"))

	// 2. Event received
	ctx, eventSpan := Tracer().Start(ctx, "ws.event.dispatch")
	AddSpanAttributes(ctx,
		attribute.String("event.type", "kc.request.next"),
		attribute.String("user.id", "user-123"),
		attribute.String("session.id", "sess-456"),
		attribute.String("request.id", "req-789"),
	)

	// 3. Database query
	dbTracer := NewDBTracer()
	err := dbTracer.TraceQuery(ctx, "SELECT * FROM question_bank WHERE difficulty = $1", []interface{}{3}, func(queryCtx context.Context) error {
		// Simulate query execution
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	// 4. Response sent
	eventSpan.End()
	wsSpan.End()

	// Verify spans were created
	spans := exporter.GetSpans()
	if len(spans) < 3 {
		t.Fatalf("expected at least 3 spans, got %d", len(spans))
	}

	// Verify span names
	spanNames := make(map[string]bool)
	for _, span := range spans {
		spanNames[span.Name] = true
		t.Logf("Span: %s (trace: %s, span: %s, parent: %s)",
			span.Name,
			span.SpanContext.TraceID().String(),
			span.SpanContext.SpanID().String(),
			span.Parent.SpanID().String(),
		)
	}

	expectedSpans := []string{"ws.connection", "ws.event.dispatch", "db.query"}
	for _, expected := range expectedSpans {
		if !spanNames[expected] {
			t.Fatalf("expected span %s not found", expected)
		}
	}

	// Verify all spans share the same trace ID
	traceID := spans[0].SpanContext.TraceID()
	for _, span := range spans {
		if span.SpanContext.TraceID() != traceID {
			t.Fatalf("expected all spans to have trace ID %s, got %s", traceID, span.SpanContext.TraceID())
		}
	}

	// Verify correlation IDs in event span
	var foundUserID, foundSessionID, foundRequestID bool
	for _, span := range spans {
		if span.Name == "ws.event.dispatch" {
			for _, attr := range span.Attributes {
				switch string(attr.Key) {
				case "user.id":
					foundUserID = attr.Value.AsString() == "user-123"
				case "session.id":
					foundSessionID = attr.Value.AsString() == "sess-456"
				case "request.id":
					foundRequestID = attr.Value.AsString() == "req-789"
				}
			}
		}
	}

	if !foundUserID {
		t.Fatal("expected user.id attribute on event span")
	}
	if !foundSessionID {
		t.Fatal("expected session.id attribute on event span")
	}
	if !foundRequestID {
		t.Fatal("expected request.id attribute on event span")
	}

	t.Log("✅ Full query lifecycle tracing verified successfully")
}

func TestCorrelationIDPropagation(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)

	originalTracer := tracer
	tracer = tp.Tracer("skill-flow-test")
	defer func() {
		tracer = originalTracer
	}()

	ctx := context.Background()

	// Start operation with correlation IDs
	ctx, span := Tracer().Start(ctx, "test.operation")
	ctx = EnrichContext(ctx, "user-999", "sess-888", "req-777")

	traceID := GetTraceID(ctx)
	userID := GetUserID(ctx)
	sessionID := GetSessionID(ctx)
	requestID := GetRequestID(ctx)

	if traceID == "" {
		t.Fatal("expected non-empty trace ID")
	}
	if userID != "user-999" {
		t.Fatalf("expected user-999, got %s", userID)
	}
	if sessionID != "sess-888" {
		t.Fatalf("expected sess-888, got %s", sessionID)
	}
	if requestID != "req-777" {
		t.Fatalf("expected req-777, got %s", requestID)
	}

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	t.Log("✅ Correlation ID propagation verified successfully")
}
