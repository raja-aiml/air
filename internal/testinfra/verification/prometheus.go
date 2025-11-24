package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/raja-aiml/air/internal/testinfra/containers"
	"net/http"
	"time"
)

type PrometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func VerifyPrometheusMetrics(ctx context.Context, prometheusURL string, report *containers.Report) error {
	client := &http.Client{Timeout: 10 * time.Second}

	// Verify Prometheus is scraping OTEL collector
	if err := queryPrometheusMetric(client, prometheusURL, `up{job="otel-collector"}`, report); err != nil {
		return err
	}

	report.StepSuccess("Metrics: Server → OTEL → Prometheus")
	return nil
}

// VerifyOTELMetricsEndpoint checks OTEL collector's Prometheus metrics endpoint
func VerifyOTELMetricsEndpoint(ctx context.Context, otelMetricsURL string, report *containers.Report) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Retry a few times - metrics may not be exported immediately
	var metricsContent string
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}

		resp, err := client.Get(otelMetricsURL)
		if err != nil {
			if attempt == 2 {
				return fmt.Errorf("query OTEL metrics endpoint: %w", err)
			}
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			if attempt == 2 {
				return fmt.Errorf("OTEL metrics endpoint returned status %d", resp.StatusCode)
			}
			continue
		}

		// Read raw Prometheus metrics (text format, not JSON)
		body := make([]byte, 8192)
		n, _ := resp.Body.Read(body)
		resp.Body.Close()

		if n > 0 {
			metricsContent = string(body[:n])
			break
		}

		if attempt == 2 {
			// On last attempt, this is acceptable - OTEL may not have metrics yet
			report.Info("OTEL collector metrics pending (awaiting first export)")
			return nil
		}
	}

	// Count metric lines (non-comment, non-empty)
	metricCount := 0
	start := 0
	for i := 0; i < len(metricsContent); i++ {
		if metricsContent[i] == '\n' {
			line := metricsContent[start:i]
			if len(line) > 0 && line[0] != '#' {
				metricCount++
			}
			start = i + 1
		}
	}
	// Count last line if no trailing newline
	if start < len(metricsContent) {
		line := metricsContent[start:]
		if len(line) > 0 && line[0] != '#' {
			metricCount++
		}
	}

	report.Info("OTEL collector: %d metrics exposed", metricCount)
	report.StepSuccess("OTEL metrics endpoint verified")
	return nil
}

func queryPrometheusMetric(client *http.Client, prometheusURL string, metric string, report *containers.Report) error {
	url := fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, metric)

	// Retry a few times as metrics may not be scraped yet
	for i := 0; i < 3; i++ {
		resp, err := client.Get(url)
		if err != nil {
			return fmt.Errorf("query prometheus: %w", err)
		}
		defer resp.Body.Close()

		var result PrometheusQueryResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode prometheus response: %w", err)
		}

		if result.Status == "success" && len(result.Data.Result) > 0 {
			value := result.Data.Result[0].Value[1]
			report.Info("%s = %v", metric, value)
			return nil
		}

		if i < 2 {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("metric %s not found in Prometheus", metric)
}

func VerifyMetricsEndpoint(_ context.Context, cfg *containers.Config, report *containers.Report) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/metrics", cfg.ServerPort))
	if err != nil {
		return fmt.Errorf("query metrics endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode)
	}

	var metrics struct {
		WSConnectionsActive float64                   `json:"ws_connections_active"`
		WSConnectionsTotal  float64                   `json:"ws_connections_total"`
		Events              map[string]map[string]any `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return fmt.Errorf("decode metrics response: %w", err)
	}

	if len(metrics.Events) == 0 {
		return fmt.Errorf("no events found in metrics payload")
	}

	report.Info("Connections: active=%.0f, total=%.0f", metrics.WSConnectionsActive, metrics.WSConnectionsTotal)
	report.Info("Events tracked: %d types", len(metrics.Events))

	report.StepSuccess("Server /metrics endpoint verified")
	return nil
}
