package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/raja-aiml/air/internal/engine"
	"github.com/raja-aiml/air/internal/foundation/httpclient"
)

// ObsCommands holds dependencies for observability commands.
type ObsCommands struct {
	jaegerURL     string
	prometheusURL string
}

// NewObsCommands creates observability command handlers.
func NewObsCommands() *ObsCommands {
	return &ObsCommands{
		jaegerURL:     "http://localhost:16686",
		prometheusURL: "http://localhost:9090",
	}
}

// Register adds all observability commands to the registry.
func (c *ObsCommands) Register(r *engine.Registry) {
	r.Register(&engine.Command{
		Name:        "obs.verify",
		Description: "Verify observability stack is healthy (Jaeger, Prometheus)",
		Examples: []string{
			"verify observability",
			"check observability health",
			"is tracing working",
			"verify jaeger and prometheus",
			"health check observability",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.verify,
	})

	r.Register(&engine.Command{
		Name:        "obs.urls",
		Description: "Show URLs for observability services",
		Examples: []string{
			"show observability urls",
			"where is jaeger",
			"prometheus url",
			"get service urls",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.urls,
	})

	r.Register(&engine.Command{
		Name:        "obs.services",
		Description: "List available services in Jaeger",
		Examples: []string{
			"list traced services",
			"show jaeger services",
			"what services are being traced",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.services,
	})

	r.Register(&engine.Command{
		Name:        "obs.metrics",
		Description: "Query Prometheus metrics",
		Examples: []string{
			"query metrics",
			"get prometheus metrics",
			"show metrics",
		},
		Parameters: []engine.Parameter{
			{Name: "query", Type: "string", Description: "PromQL query (default: up)"},
		},
		Execute: c.metrics,
	})
}

func (c *ObsCommands) verify(ctx context.Context, params map[string]any) (engine.Result, error) {
	results := make(map[string]string)

	// Check Jaeger
	jaegerOK := c.checkEndpoint(ctx, c.jaegerURL+"/api/services")
	if jaegerOK {
		results["jaeger"] = "healthy"
	} else {
		results["jaeger"] = "unhealthy"
	}

	// Check Prometheus
	prometheusOK := c.checkEndpoint(ctx, c.prometheusURL+"/-/healthy")
	if prometheusOK {
		results["prometheus"] = "healthy"
	} else {
		results["prometheus"] = "unhealthy"
	}

	allHealthy := jaegerOK && prometheusOK
	message := "Observability Stack Status:\n"
	for svc, status := range results {
		icon := "+"
		if status == "unhealthy" {
			icon = "x"
		}
		message += fmt.Sprintf("  %s %s: %s\n", icon, svc, status)
	}

	if allHealthy {
		message += "\nAll observability services are healthy!"
	} else {
		message += "\nSome services are unhealthy. Run 'air infra start' to start them."
	}

	return engine.NewResultWithData(message, results), nil
}

func (c *ObsCommands) urls(ctx context.Context, params map[string]any) (engine.Result, error) {
	urls := map[string]string{
		"jaeger":     c.jaegerURL,
		"prometheus": c.prometheusURL,
	}

	message := "Observability Service URLs:\n"
	message += fmt.Sprintf("  Jaeger UI:     %s\n", c.jaegerURL)
	message += fmt.Sprintf("  Prometheus:    %s\n", c.prometheusURL)

	return engine.NewResultWithData(message, urls), nil
}

func (c *ObsCommands) services(ctx context.Context, params map[string]any) (engine.Result, error) {
	client := httpclient.Default()

	var result struct {
		Data []string `json:"data"`
	}
	if err := client.GetJSON(ctx, c.jaegerURL+"/api/services", &result); err != nil {
		err = fmt.Errorf("failed to connect to Jaeger: %w", err)
		return engine.ErrorResult(err), err
	}

	message := "Traced Services in Jaeger:\n"
	if len(result.Data) == 0 {
		message += "  (no services found - run your application to generate traces)"
	} else {
		for _, svc := range result.Data {
			message += fmt.Sprintf("  - %s\n", svc)
		}
	}

	return engine.NewResultWithData(message, result.Data), nil
}

func (c *ObsCommands) metrics(ctx context.Context, params map[string]any) (engine.Result, error) {
	p := engine.Params(params)
	query := p.String("query", "up")

	client := httpclient.Default()

	url := fmt.Sprintf("%s/api/v1/query?query=%s", c.prometheusURL, query)
	var result map[string]interface{}
	if err := client.GetJSON(ctx, url, &result); err != nil {
		err = fmt.Errorf("failed to connect to Prometheus: %w", err)
		return engine.ErrorResult(err), err
	}

	// Pretty print the result
	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	message := fmt.Sprintf("Prometheus Query: %s\n\n%s", query, string(prettyJSON))

	return engine.NewResultWithData(message, result), nil
}

func (c *ObsCommands) checkEndpoint(ctx context.Context, url string) bool {
	return httpclient.Default().CheckEndpoint(ctx, url)
}
