package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Metrics collects application metrics for observability.
type Metrics struct {
	mu                  sync.RWMutex
	wsConnectionsActive int64
	wsConnectionsTotal  int64
	wsEventsProcessed   map[string]int64
	wsEventErrors       map[string]int64
	wsEventLatency      map[string][]time.Duration
}

var globalMetrics = &Metrics{
	wsEventsProcessed: make(map[string]int64),
	wsEventErrors:     make(map[string]int64),
	wsEventLatency:    make(map[string][]time.Duration),
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	return globalMetrics
}

// WSConnectionOpened increments active connection count.
func (m *Metrics) WSConnectionOpened() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsConnectionsActive++
	m.wsConnectionsTotal++
	log.Info().Int64("active", m.wsConnectionsActive).Int64("total", m.wsConnectionsTotal).Msg("ws connection opened")
}

// WSConnectionClosed decrements active connection count.
func (m *Metrics) WSConnectionClosed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsConnectionsActive--
	log.Info().Int64("active", m.wsConnectionsActive).Msg("ws connection closed")
}

// WSEventProcessed records a successfully processed event.
func (m *Metrics) WSEventProcessed(eventName string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsEventsProcessed[eventName]++
	m.wsEventLatency[eventName] = append(m.wsEventLatency[eventName], duration)
	if len(m.wsEventLatency[eventName]) > 100 {
		m.wsEventLatency[eventName] = m.wsEventLatency[eventName][1:]
	}
}

// WSEventError records an event processing error.
func (m *Metrics) WSEventError(eventName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsEventErrors[eventName]++
	log.Warn().Str("event", eventName).Int64("total_errors", m.wsEventErrors[eventName]).Msg("ws event error")
}

// GetStats returns current metrics snapshot.
func (m *Metrics) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	eventStats := make(map[string]EventStats)
	for event, count := range m.wsEventsProcessed {
		avg := time.Duration(0)
		if len(m.wsEventLatency[event]) > 0 {
			var sum time.Duration
			for _, d := range m.wsEventLatency[event] {
				sum += d
			}
			avg = sum / time.Duration(len(m.wsEventLatency[event]))
		}
		eventStats[event] = EventStats{
			Count:          count,
			Errors:         m.wsEventErrors[event],
			AvgLatency:     avg,
			LatencySamples: len(m.wsEventLatency[event]),
		}
	}

	return Stats{
		WSConnectionsActive: m.wsConnectionsActive,
		WSConnectionsTotal:  m.wsConnectionsTotal,
		EventStats:          eventStats,
	}
}

// Stats is a snapshot of metrics.
type Stats struct {
	WSConnectionsActive int64
	WSConnectionsTotal  int64
	EventStats          map[string]EventStats
}

// EventStats contains metrics for a specific event type.
type EventStats struct {
	Count          int64
	Errors         int64
	AvgLatency     time.Duration
	LatencySamples int
}

// Reset clears all metrics (useful for testing).
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsConnectionsActive = 0
	m.wsConnectionsTotal = 0
	m.wsEventsProcessed = make(map[string]int64)
	m.wsEventErrors = make(map[string]int64)
	m.wsEventLatency = make(map[string][]time.Duration)
}

// Convenience helpers for global metrics.
func IncWS() {
	globalMetrics.WSConnectionOpened()
}

func DecWS() {
	globalMetrics.WSConnectionClosed()
}

// MetricsHandler renders a minimal Prometheus-style payload.
func MetricsHandler() []byte {
	stats := globalMetrics.GetStats()
	latency := time.Duration(0)
	if eStats, ok := stats.EventStats["kc.request.next"]; ok {
		latency = eStats.AvgLatency
	}
	return []byte(fmt.Sprintf("# TYPE ws_connections gauge\nws_connections %d\n# TYPE ws_events_total counter\nws_events_total %d\n# TYPE ws_kc_request_next_latency_seconds gauge\nws_kc_request_next_latency_seconds %.6f\n",
		stats.WSConnectionsActive,
		totalEvents(stats.EventStats),
		latency.Seconds()))
}

func totalEvents(es map[string]EventStats) int64 {
	var total int64
	for _, s := range es {
		total += s.Count
	}
	return total
}
