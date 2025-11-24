package telemetry

import (
	"context"
	"testing"
)

func TestCorrelationID(t *testing.T) {
	ctx := context.Background()
	correlationID := "test-correlation-123"

	ctx = WithCorrelationID(ctx, correlationID)
	retrieved := GetCorrelationID(ctx)

	if retrieved != correlationID {
		t.Fatalf("expected %s, got %s", correlationID, retrieved)
	}
}

func TestRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "req-456"

	ctx = WithRequestID(ctx, requestID)
	retrieved := GetRequestID(ctx)

	if retrieved != requestID {
		t.Fatalf("expected %s, got %s", requestID, retrieved)
	}
}

func TestUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user-789"

	ctx = WithUserID(ctx, userID)
	retrieved := GetUserID(ctx)

	if retrieved != userID {
		t.Fatalf("expected %s, got %s", userID, retrieved)
	}
}

func TestSessionID(t *testing.T) {
	ctx := context.Background()
	sessionID := "sess-012"

	ctx = WithSessionID(ctx, sessionID)
	retrieved := GetSessionID(ctx)

	if retrieved != sessionID {
		t.Fatalf("expected %s, got %s", sessionID, retrieved)
	}
}

func TestNewCorrelationID(t *testing.T) {
	id1 := NewCorrelationID()
	id2 := NewCorrelationID()

	if id1 == "" {
		t.Fatal("expected non-empty correlation ID")
	}
	if id2 == "" {
		t.Fatal("expected non-empty correlation ID")
	}
	if id1 == id2 {
		t.Fatal("expected unique correlation IDs")
	}
}

func TestEnrichContext(t *testing.T) {
	ctx := context.Background()
	userID := "user-123"
	sessionID := "sess-456"
	requestID := "req-789"

	enriched := EnrichContext(ctx, userID, sessionID, requestID)

	if GetUserID(enriched) != userID {
		t.Fatalf("expected user ID %s, got %s", userID, GetUserID(enriched))
	}
	if GetSessionID(enriched) != sessionID {
		t.Fatalf("expected session ID %s, got %s", sessionID, GetSessionID(enriched))
	}
	if GetRequestID(enriched) != requestID {
		t.Fatalf("expected request ID %s, got %s", requestID, GetRequestID(enriched))
	}
}

func TestEnrichContextGeneratesRequestID(t *testing.T) {
	ctx := context.Background()
	enriched := EnrichContext(ctx, "user-123", "sess-456", "")

	requestID := GetRequestID(enriched)
	if requestID == "" {
		t.Fatal("expected auto-generated request ID")
	}
}

func TestGetMissingValues(t *testing.T) {
	ctx := context.Background()

	if GetCorrelationID(ctx) != "" {
		t.Fatal("expected empty correlation ID")
	}
	if GetRequestID(ctx) != "" {
		t.Fatal("expected empty request ID")
	}
	if GetUserID(ctx) != "" {
		t.Fatal("expected empty user ID")
	}
	if GetSessionID(ctx) != "" {
		t.Fatal("expected empty session ID")
	}
}
