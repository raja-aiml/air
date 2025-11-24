package metrics

import (
	"testing"
	"time"
)

func TestWSConnectionOpened(t *testing.T) {
	m := &Metrics{
		wsEventsProcessed: make(map[string]int64),
		wsEventErrors:     make(map[string]int64),
		wsEventLatency:    make(map[string][]time.Duration),
	}

	m.WSConnectionOpened()

	stats := m.GetStats()
	if stats.WSConnectionsActive != 1 {
		t.Fatalf("expected active connections = 1, got %d", stats.WSConnectionsActive)
	}
	if stats.WSConnectionsTotal != 1 {
		t.Fatalf("expected total connections = 1, got %d", stats.WSConnectionsTotal)
	}

	m.WSConnectionOpened()
	stats = m.GetStats()
	if stats.WSConnectionsActive != 2 {
		t.Fatalf("expected active connections = 2, got %d", stats.WSConnectionsActive)
	}
	if stats.WSConnectionsTotal != 2 {
		t.Fatalf("expected total connections = 2, got %d", stats.WSConnectionsTotal)
	}
}

func TestWSConnectionClosed(t *testing.T) {
	m := &Metrics{
		wsEventsProcessed: make(map[string]int64),
		wsEventErrors:     make(map[string]int64),
		wsEventLatency:    make(map[string][]time.Duration),
	}

	m.WSConnectionOpened()
	m.WSConnectionOpened()
	m.WSConnectionClosed()

	stats := m.GetStats()
	if stats.WSConnectionsActive != 1 {
		t.Fatalf("expected active connections = 1, got %d", stats.WSConnectionsActive)
	}
}

func TestWSEventProcessed(t *testing.T) {
	m := &Metrics{
		wsEventsProcessed: make(map[string]int64),
		wsEventErrors:     make(map[string]int64),
		wsEventLatency:    make(map[string][]time.Duration),
	}

	eventName := "kc.request.next"
	duration := 100 * time.Millisecond

	m.WSEventProcessed(eventName, duration)
	m.WSEventProcessed(eventName, duration)

	stats := m.GetStats()
	eventStats, ok := stats.EventStats[eventName]
	if !ok {
		t.Fatalf("expected event stats for %s", eventName)
	}
	if eventStats.Count != 2 {
		t.Fatalf("expected 2 processed events, got %d", eventStats.Count)
	}

	// Check latency recorded
	if eventStats.LatencySamples != 2 {
		t.Fatalf("expected 2 latency samples, got %d", eventStats.LatencySamples)
	}
}

func TestWSEventError(t *testing.T) {
	m := &Metrics{
		wsEventsProcessed: make(map[string]int64),
		wsEventErrors:     make(map[string]int64),
		wsEventLatency:    make(map[string][]time.Duration),
	}

	eventName := "kc.answer.submit"

	// Need at least one processed event for stats to appear
	m.WSEventProcessed(eventName, 10*time.Millisecond)
	m.WSEventError(eventName)
	m.WSEventError(eventName)
	m.WSEventError(eventName)

	stats := m.GetStats()
	eventStats, ok := stats.EventStats[eventName]
	if !ok {
		t.Fatalf("expected event stats for %s", eventName)
	}
	if eventStats.Errors != 3 {
		t.Fatalf("expected 3 error events, got %d", eventStats.Errors)
	}
}

func TestGetStats(t *testing.T) {
	m := &Metrics{
		wsEventsProcessed: make(map[string]int64),
		wsEventErrors:     make(map[string]int64),
		wsEventLatency:    make(map[string][]time.Duration),
	}

	m.WSConnectionOpened()
	m.WSEventProcessed("test.event", 50*time.Millisecond)
	m.WSEventError("test.event") // Error for same event

	stats := m.GetStats()
	if stats.WSConnectionsActive != 1 {
		t.Fatalf("expected active = 1, got %d", stats.WSConnectionsActive)
	}
	if stats.EventStats["test.event"].Count != 1 {
		t.Fatalf("expected 1 processed event, got %d", stats.EventStats["test.event"].Count)
	}
	if stats.EventStats["test.event"].Errors != 1 {
		t.Fatalf("expected 1 error, got %d", stats.EventStats["test.event"].Errors)
	}
}

func TestReset(t *testing.T) {
	m := &Metrics{
		wsEventsProcessed: make(map[string]int64),
		wsEventErrors:     make(map[string]int64),
		wsEventLatency:    make(map[string][]time.Duration),
	}

	m.WSConnectionOpened()
	m.WSEventProcessed("test", 10*time.Millisecond)
	m.WSEventError("test")

	m.Reset()

	stats := m.GetStats()
	if stats.WSConnectionsActive != 0 {
		t.Fatalf("expected active = 0 after reset, got %d", stats.WSConnectionsActive)
	}
	if stats.WSConnectionsTotal != 0 {
		t.Fatalf("expected total = 0 after reset, got %d", stats.WSConnectionsTotal)
	}
	if len(stats.EventStats) != 0 {
		t.Fatalf("expected empty event stats after reset, got %d", len(stats.EventStats))
	}
}

func TestGetMetrics(t *testing.T) {
	m := GetMetrics()
	if m == nil {
		t.Fatal("expected non-nil global metrics")
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := &Metrics{
		wsEventsProcessed: make(map[string]int64),
		wsEventErrors:     make(map[string]int64),
		wsEventLatency:    make(map[string][]time.Duration),
	}

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			m.WSConnectionOpened()
			m.WSEventProcessed("test", 10*time.Millisecond)
			m.WSConnectionClosed()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and have consistent state
	stats := m.GetStats()
	if stats.WSConnectionsTotal != 10 {
		t.Fatalf("expected total connections = 10, got %d", stats.WSConnectionsTotal)
	}
}
