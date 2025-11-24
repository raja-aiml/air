package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupDBTestTracer(_ *testing.T) (*tracetest.InMemoryExporter, func()) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("skill-flow")

	cleanup := func() {
		exporter.Reset()
	}

	return exporter, cleanup
}

func TestNewDBTracer(t *testing.T) {
	dbTracer := NewDBTracer()
	if dbTracer == nil {
		t.Fatal("expected non-nil DBTracer")
	}
}

func TestTraceQuerySuccess(t *testing.T) {
	exporter, cleanup := setupDBTestTracer(t)
	defer cleanup()

	ctx := context.Background()
	tr := Tracer()
	ctx, parentSpan := tr.Start(ctx, "parent")
	defer parentSpan.End()

	dbTracer := NewDBTracer()
	query := "SELECT * FROM users WHERE id = $1"
	args := []interface{}{123}

	executed := false
	err := dbTracer.TraceQuery(ctx, query, args, func(queryCtx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !executed {
		t.Fatal("expected query function to be executed")
	}

	spans := exporter.GetSpans()
	if len(spans) < 1 {
		t.Fatalf("expected at least 1 span, got %d", len(spans))
	}

	// Find db.query span
	var dbSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "db.query" {
			dbSpan = &spans[i]
			break
		}
	}

	if dbSpan == nil {
		t.Fatal("expected to find db.query span")
	}

	// Verify attributes
	attrMap := make(map[string]interface{})
	for _, attr := range dbSpan.Attributes {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	if attrMap["db.system"] != "postgresql" {
		t.Fatalf("expected db.system='postgresql', got '%v'", attrMap["db.system"])
	}
	if attrMap["db.statement"] != query {
		t.Fatalf("expected db.statement='%s', got '%v'", query, attrMap["db.statement"])
	}

	// Verify status is OK
	if dbSpan.Status.Code != codes.Ok {
		t.Fatalf("expected OK status, got %v", dbSpan.Status.Code)
	}
}

func TestTraceQueryError(t *testing.T) {
	exporter, cleanup := setupDBTestTracer(t)
	defer cleanup()

	ctx := context.Background()
	tr := Tracer()
	ctx, parentSpan := tr.Start(ctx, "parent")
	defer parentSpan.End()

	dbTracer := NewDBTracer()
	query := "SELECT * FROM nonexistent"
	testErr := errors.New("table does not exist")

	err := dbTracer.TraceQuery(ctx, query, nil, func(queryCtx context.Context) error {
		return testErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != testErr {
		t.Fatalf("expected error '%v', got '%v'", testErr, err)
	}

	spans := exporter.GetSpans()

	// Find db.query span
	var dbSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "db.query" {
			dbSpan = &spans[i]
			break
		}
	}

	if dbSpan == nil {
		t.Fatal("expected to find db.query span")
	}

	// Verify error status
	if dbSpan.Status.Code != codes.Error {
		t.Fatalf("expected Error status, got %v", dbSpan.Status.Code)
	}

	// Verify error event
	if len(dbSpan.Events) == 0 {
		t.Fatal("expected error event in span")
	}
}

// TestTraceTransaction requires a real pgx.Tx, so we skip it in unit tests
// It should be tested in integration tests with a real database connection

func TestTraceMigration(t *testing.T) {
	exporter, cleanup := setupDBTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	dbTracer := NewDBTracer()
	migrationVersion := 1

	executed := false
	err := dbTracer.TraceMigration(ctx, migrationVersion, func(migCtx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !executed {
		t.Fatal("expected migration function to be executed")
	}

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least 1 span")
	}

	// Find db.migration span
	var migSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "db.migration" {
			migSpan = &spans[i]
			break
		}
	}

	if migSpan == nil {
		t.Fatal("expected to find db.migration span")
	}

	// Verify attributes
	attrMap := make(map[string]interface{})
	for _, attr := range migSpan.Attributes {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	if attrMap["db.migration.version"] != int64(migrationVersion) {
		t.Fatalf("expected db.migration.version=%d, got '%v'", migrationVersion, attrMap["db.migration.version"])
	}

	// Verify status is OK
	if migSpan.Status.Code != codes.Ok {
		t.Fatalf("expected OK status, got %v", migSpan.Status.Code)
	}
}
