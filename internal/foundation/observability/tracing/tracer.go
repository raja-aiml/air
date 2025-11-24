package telemetry

import (
	"context"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var tracer trace.Tracer = otel.Tracer("skill-flow")

// InitTracer initializes OpenTelemetry tracer from environment variables
func InitTracer(ctx context.Context) (func(context.Context) error, error) {
	enabled, _ := strconv.ParseBool(os.Getenv("OTEL_ENABLED"))
	if !enabled {
		// Return no-op shutdown
		return func(context.Context) error { return nil }, nil
	}

	endpoint := os.Getenv("OTEL_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "skillflow-backend"
	}

	environment := os.Getenv("OTEL_ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, err
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create tracer provider
	// Use syncer instead of batcher for immediate export (useful for short-lived processes/tests)
	useSyncer, _ := strconv.ParseBool(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_SYNC"))
	var spanProcessor sdktrace.SpanProcessor
	if useSyncer {
		spanProcessor = sdktrace.NewSimpleSpanProcessor(exporter)
	} else {
		spanProcessor = sdktrace.NewBatchSpanProcessor(exporter)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanProcessor),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("skill-flow")

	// Return shutdown function
	return tp.Shutdown, nil
}

// Tracer returns the global tracer (noop by default).
func Tracer() trace.Tracer {
	return tracer
}

// GetTraceID returns the current trace ID from context if present.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	sc := span.SpanContext()
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}

// AddSpanAttributes adds attributes to the current span if any.
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	span.SetAttributes(attrs...)
}

// LogInfo adds an info-level log event to the current span.
// These appear in Jaeger UI under the span's "Logs" tab.
func LogInfo(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}
	attrs = append(attrs, attribute.String("level", "info"))
	span.AddEvent(message, trace.WithAttributes(attrs...))
}

// LogDebug adds a debug-level log event to the current span.
func LogDebug(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}
	attrs = append(attrs, attribute.String("level", "debug"))
	span.AddEvent(message, trace.WithAttributes(attrs...))
}

// LogWarn adds a warning-level log event to the current span.
func LogWarn(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}
	attrs = append(attrs, attribute.String("level", "warn"))
	span.AddEvent(message, trace.WithAttributes(attrs...))
}

// LogError adds an error-level log event to the current span.
// It also records the error on the span.
func LogError(ctx context.Context, message string, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}
	attrs = append(attrs, attribute.String("level", "error"))
	if err != nil {
		attrs = append(attrs, attribute.String("error.message", err.Error()))
		span.RecordError(err)
	}
	span.AddEvent(message, trace.WithAttributes(attrs...))
}

// LogEvent adds a custom event to the current span with optional attributes.
// Use this for domain-specific events that don't fit standard log levels.
func LogEvent(ctx context.Context, eventName string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}
