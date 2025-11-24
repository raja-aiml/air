package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracer(_ *testing.T) (*tracetest.InMemoryExporter, func()) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("skill-flow")

	cleanup := func() {
		exporter.Reset()
	}

	return exporter, cleanup
}

func TestTracer(t *testing.T) {
	tr := Tracer()
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestGetTraceID(t *testing.T) {
	exporter, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()
	tr := Tracer()
	ctx, span := tr.Start(ctx, "test.traceid")

	traceID := GetTraceID(ctx)
	if traceID == "" {
		t.Fatal("expected non-empty trace ID")
	}

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	expectedTraceID := spans[0].SpanContext.TraceID().String()
	if traceID != expectedTraceID {
		t.Fatalf("expected trace ID '%s', got '%s'", expectedTraceID, traceID)
	}
}

func TestGetTraceIDNoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := GetTraceID(ctx)
	if traceID != "" {
		t.Fatalf("expected empty trace ID for context without span, got '%s'", traceID)
	}
}

func TestAddSpanAttributes(t *testing.T) {
	exporter, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()
	tr := Tracer()
	ctx, span := tr.Start(ctx, "test.attributes")

	AddSpanAttributes(ctx,
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
		attribute.Bool("key3", true),
	)

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	attrs := spans[0].Attributes
	attrMap := make(map[string]interface{})
	for _, attr := range attrs {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	if attrMap["key1"] != "value1" {
		t.Fatalf("expected key1='value1', got '%v'", attrMap["key1"])
	}
	if attrMap["key2"] != int64(42) {
		t.Fatalf("expected key2=42, got %v", attrMap["key2"])
	}
	if attrMap["key3"] != true {
		t.Fatalf("expected key3=true, got %v", attrMap["key3"])
	}
}

func TestAddSpanAttributesNoSpan(t *testing.T) {
	ctx := context.Background()
	// Should not panic
	AddSpanAttributes(ctx,
		attribute.String("key1", "value1"),
	)
}

func TestNestedSpans(t *testing.T) {
	exporter, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()
	tr := Tracer()
	parentCtx, parentSpan := tr.Start(ctx, "parent")
	_, childSpan := tr.Start(parentCtx, "child")

	childSpan.End()
	parentSpan.End()

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Find parent and child
	var parent, child *tracetest.SpanStub
	for i := range spans {
		switch spans[i].Name {
		case "parent":
			parent = &spans[i]
		case "child":
			child = &spans[i]
		}
	}

	if parent == nil || child == nil {
		t.Fatal("expected both parent and child spans")
	}

	// Verify parent-child relationship
	if child.Parent.SpanID() != parent.SpanContext.SpanID() {
		t.Fatal("expected child span to have parent as parent")
	}

	// Verify same trace ID
	if child.SpanContext.TraceID() != parent.SpanContext.TraceID() {
		t.Fatal("expected parent and child to share same trace ID")
	}
}
