package telemetry

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DBTracer wraps database operations with tracing
type DBTracer struct {
	tracer trace.Tracer
}

// NewDBTracer creates a new database tracer
func NewDBTracer() *DBTracer {
	return &DBTracer{
		tracer: Tracer(),
	}
}

// TraceQuery wraps a database query with tracing
func (t *DBTracer) TraceQuery(ctx context.Context, query string, args []interface{}, fn func(context.Context) error) error {
	ctx, span := t.tracer.Start(ctx, "db.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.statement", query),
		),
	)
	defer span.End()

	// Add query parameters (sanitized)
	if len(args) > 0 {
		span.SetAttributes(attribute.Int("db.params.count", len(args)))
	}

	// Log query execution start (appears in Jaeger's Logs tab)
	LogDebug(ctx, "executing query", attribute.Int("param_count", len(args)))

	err := fn(ctx)
	if err != nil {
		LogError(ctx, "query failed", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	LogInfo(ctx, "query completed successfully")
	span.SetStatus(codes.Ok, "query successful")
	return nil
}

// TraceTransaction wraps a database transaction with tracing
func (t *DBTracer) TraceTransaction(ctx context.Context, name string, fn func(context.Context, pgx.Tx) error, tx pgx.Tx) error {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("db.transaction.%s", name),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "transaction"),
		),
	)
	defer span.End()

	LogInfo(ctx, "transaction started", attribute.String("transaction.name", name))

	err := fn(ctx, tx)
	if err != nil {
		LogError(ctx, "transaction failed", err, attribute.String("transaction.name", name))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	LogInfo(ctx, "transaction committed", attribute.String("transaction.name", name))
	span.SetStatus(codes.Ok, "transaction successful")
	return nil
}

// TraceMigration wraps database migration with tracing
func (t *DBTracer) TraceMigration(ctx context.Context, version int, fn func(context.Context) error) error {
	ctx, span := t.tracer.Start(ctx, "db.migration",
		trace.WithAttributes(
			attribute.Int("db.migration.version", version),
		),
	)
	defer span.End()

	LogInfo(ctx, "migration started", attribute.Int("version", version))

	err := fn(ctx)
	if err != nil {
		LogError(ctx, "migration failed", err, attribute.Int("version", version))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	LogInfo(ctx, "migration completed", attribute.Int("version", version))
	span.SetStatus(codes.Ok, "migration successful")
	return nil
}
