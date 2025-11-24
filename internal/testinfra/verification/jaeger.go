package verification

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/raja-aiml/air/internal/testinfra/containers"
	"net/http"
	"net/url"
	"time"
)

type JaegerTrace struct {
	Data []struct {
		TraceID string `json:"traceID"`
		Spans   []struct {
			TraceID       string `json:"traceID"`
			SpanID        string `json:"spanID"`
			OperationName string `json:"operationName"`
			References    []struct {
				RefType string `json:"refType"`
				TraceID string `json:"traceID"`
				SpanID  string `json:"spanID"`
			} `json:"references"`
			StartTime int64 `json:"startTime"`
			Duration  int64 `json:"duration"`
			Tags      []struct {
				Key   string      `json:"key"`
				Type  string      `json:"type"`
				Value interface{} `json:"value"`
			} `json:"tags"`
		} `json:"spans"`
	} `json:"data"`
}

func VerifyJaegerTraces(_ context.Context, cfg *containers.Config, jaegerURL string, correlationIDs map[string]string, report *containers.Report) error {
	report.Step("Querying Jaeger for trace...")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Build Jaeger query - search by service and filter client-side
	// Use wide time range and high limit to ensure we get recent traces
	query := fmt.Sprintf("%s/api/traces?service=%s&lookback=5m&limit=100",
		jaegerURL, url.QueryEscape(cfg.ServiceName))

	// Retry logic: wait for traces to propagate through OTEL collector to Jaeger
	var trace JaegerTrace
	maxAttempts := 10
	retryDelay := 500 * time.Millisecond

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Silent retry
			time.Sleep(retryDelay)
		}

		resp, err := client.Get(query)
		if err != nil {
			if attempt == maxAttempts {
				return fmt.Errorf("query jaeger: %w", err)
			}
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			if attempt == maxAttempts {
				return fmt.Errorf("jaeger returned status %d", resp.StatusCode)
			}
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&trace); err != nil {
			resp.Body.Close()
			if attempt == maxAttempts {
				return fmt.Errorf("decode jaeger response: %w", err)
			}
			continue
		}
		resp.Body.Close()

		// Filter traces by correlation IDs (client-side filtering)
		if len(trace.Data) > 0 {
			// Check if any trace has matching correlation IDs
			for _, traceData := range trace.Data {
				for _, span := range traceData.Spans {
					matchingTags := 0
					for _, tag := range span.Tags {
						if tag.Key == "user.id" && fmt.Sprint(tag.Value) == correlationIDs["user_id"] {
							matchingTags++
						}
						if tag.Key == "session.id" && fmt.Sprint(tag.Value) == correlationIDs["session_id"] {
							matchingTags++
						}
						if tag.Key == "request.id" && fmt.Sprint(tag.Value) == correlationIDs["request_id"] {
							matchingTags++
						}
					}
					// If we found at least 2 matching tags, consider this a match
					if matchingTags >= 2 {
						// Re-structure trace.Data to only include matching trace
						trace.Data = []struct {
							TraceID string `json:"traceID"`
							Spans   []struct {
								TraceID       string `json:"traceID"`
								SpanID        string `json:"spanID"`
								OperationName string `json:"operationName"`
								References    []struct {
									RefType string `json:"refType"`
									TraceID string `json:"traceID"`
									SpanID  string `json:"spanID"`
								} `json:"references"`
								StartTime int64 `json:"startTime"`
								Duration  int64 `json:"duration"`
								Tags      []struct {
									Key   string      `json:"key"`
									Type  string      `json:"type"`
									Value interface{} `json:"value"`
								} `json:"tags"`
							} `json:"spans"`
						}{traceData}
						goto found
					}
				}
			}
		}

		if attempt == maxAttempts {
			return fmt.Errorf("no trace found for correlation IDs %v after %d attempts", correlationIDs, maxAttempts)
		}
		continue
	found:
		break
	}

	if len(trace.Data) == 0 {
		return fmt.Errorf("no trace found for correlation IDs %v", correlationIDs)
	}

	spans := trace.Data[0].Spans

	// Verify expected spans exist
	expectedSpans := []string{"ws.connection", "ws.auth", "ws.event.dispatch", "db.query"}
	foundSpans := make(map[string]bool)
	spanNames := make([]string, 0, len(spans))

	for _, span := range spans {
		foundSpans[span.OperationName] = true
		spanNames = append(spanNames, span.OperationName)
	}

	report.Info("Trace ID: %s (%d spans)", trace.Data[0].TraceID, len(spans))

	for _, expected := range expectedSpans {
		if !foundSpans[expected] {
			return fmt.Errorf("expected span '%s' not found", expected)
		}
	}

	// Verify correlation IDs exist in at least one span
	foundMatchingSpan := false
	for _, span := range spans {
		matchCount := 0
		for _, tag := range span.Tags {
			tagValue := fmt.Sprint(tag.Value)
			if tag.Key == "user.id" && tagValue == correlationIDs["user_id"] {
				matchCount++
			}
			if tag.Key == "session.id" && tagValue == correlationIDs["session_id"] {
				matchCount++
			}
			if tag.Key == "request.id" && tagValue == correlationIDs["request_id"] {
				matchCount++
			}
		}
		// If at least 2 out of 3 correlation IDs match, consider it found
		if matchCount >= 2 {
			report.Info("✓ Found span '%s' with matching correlation IDs (%d/3)", span.OperationName, matchCount)
			foundMatchingSpan = true
			break
		}
	}

	if !foundMatchingSpan {
		return fmt.Errorf("correlation IDs %v not found in any span", correlationIDs)
	}

	report.Info("Correlation IDs verified")
	report.StepSuccess("Traces: Server → OTEL → Jaeger")
	return nil
}
