package telemetry

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	requestIDKey     contextKey = "request_id"
	userIDKey        contextKey = "user_id"
	sessionIDKey     contextKey = "session_id"
)

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// GetCorrelationID retrieves the correlation ID from the context
func GetCorrelationID(ctx context.Context) string {
	if v := ctx.Value(correlationIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) string {
	if v := ctx.Value(userIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// WithSessionID adds a session ID to the context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// GetSessionID retrieves the session ID from the context
func GetSessionID(ctx context.Context) string {
	if v := ctx.Value(sessionIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// NewCorrelationID generates a new correlation ID
func NewCorrelationID() string {
	return uuid.NewString()
}

// EnrichContext adds all correlation IDs and trace info to the context
func EnrichContext(ctx context.Context, userID, sessionID, requestID string) context.Context {
	if requestID == "" {
		requestID = NewCorrelationID()
	}

	// Add correlation IDs
	ctx = WithRequestID(ctx, requestID)
	ctx = WithCorrelationID(ctx, GetTraceID(ctx))

	if userID != "" {
		ctx = WithUserID(ctx, userID)
	}
	if sessionID != "" {
		ctx = WithSessionID(ctx, sessionID)
	}

	// Add to span attributes
	AddSpanAttributes(ctx,
		attribute.String("request.id", requestID),
		attribute.String("user.id", userID),
		attribute.String("session.id", sessionID),
	)

	return ctx
}
