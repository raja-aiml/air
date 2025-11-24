// Package commands provides command implementations for the air CLI.
package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/raja-aiml/air/internal/engine"
	"github.com/raja-aiml/air/internal/foundation/compose"
)

// InfraCommands holds dependencies for infrastructure commands.
type InfraCommands struct {
	composeSvc *compose.Service
}

// NewInfraCommands creates infrastructure command handlers.
func NewInfraCommands(composeSvc *compose.Service) *InfraCommands {
	return &InfraCommands{composeSvc: composeSvc}
}

// Register adds all infrastructure commands to the registry.
func (c *InfraCommands) Register(r *engine.Registry) {
	r.Register(&engine.Command{
		Name:        "infra.start",
		Description: "Start infrastructure services (postgres, jaeger, prometheus, otel-collector)",
		Examples: []string{
			"start infrastructure",
			"start the services",
			"bring up infrastructure",
			"start postgres and jaeger",
			"spin up the database",
			"launch the stack",
		},
		Parameters: []engine.Parameter{
			{Name: "timeout", Type: "duration", Default: 2 * time.Minute, Description: "Timeout for health checks"},
		},
		Execute: c.start,
	})

	r.Register(&engine.Command{
		Name:        "infra.stop",
		Description: "Stop infrastructure services",
		Examples: []string{
			"stop infrastructure",
			"stop the services",
			"bring down infrastructure",
			"shut down the stack",
			"stop everything",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.stop,
	})

	r.Register(&engine.Command{
		Name:        "infra.status",
		Description: "Show status of infrastructure services",
		Examples: []string{
			"show infrastructure status",
			"what services are running",
			"check service status",
			"are services healthy",
			"list running services",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.status,
	})

	r.Register(&engine.Command{
		Name:        "infra.logs",
		Description: "Show logs from infrastructure services",
		Examples: []string{
			"show logs",
			"show postgres logs",
			"get jaeger logs",
			"view service logs",
		},
		Parameters: []engine.Parameter{
			{Name: "service", Type: "string", Description: "Service name (postgres, jaeger, prometheus, otel-collector)"},
		},
		Execute: c.logs,
	})

	r.Register(&engine.Command{
		Name:        "infra.clean",
		Description: "Remove all infrastructure containers, volumes, and networks",
		Examples: []string{
			"clean infrastructure",
			"remove all containers",
			"destroy infrastructure",
			"clean up everything",
		},
		Parameters: []engine.Parameter{},
		Execute:    c.clean,
	})
}

func (c *InfraCommands) start(ctx context.Context, params map[string]any) (engine.Result, error) {
	p := engine.Params(params)
	timeout := p.Duration("timeout", 2*time.Minute)

	if err := c.composeSvc.Start(ctx); err != nil {
		return engine.ErrorResult(err), err
	}

	// Wait for services to be healthy
	if err := c.composeSvc.WaitForHealthy(ctx, timeout); err != nil {
		return engine.Result{
			Success: true,
			Message: fmt.Sprintf("Services started but health check failed: %v", err),
		}, nil
	}

	return engine.NewResult("Infrastructure started successfully"), nil
}

func (c *InfraCommands) stop(ctx context.Context, params map[string]any) (engine.Result, error) {
	if err := c.composeSvc.Stop(ctx); err != nil {
		return engine.ErrorResult(err), err
	}
	return engine.NewResult("Infrastructure stopped"), nil
}

func (c *InfraCommands) status(ctx context.Context, params map[string]any) (engine.Result, error) {
	status, err := c.composeSvc.Status(ctx)
	if err != nil {
		return engine.ErrorResult(err), err
	}

	// Format status for display
	var sb strings.Builder
	sb.WriteString("Infrastructure Status:\n")
	for name, info := range status.Services {
		healthIcon := "?"
		switch info.Health {
		case "healthy":
			healthIcon = "+"
		case "unhealthy":
			healthIcon = "x"
		case "starting":
			healthIcon = "~"
		}
		sb.WriteString(fmt.Sprintf("  %s %s: %s (health: %s)\n", healthIcon, name, info.State, info.Health))
		if len(info.Ports) > 0 {
			sb.WriteString(fmt.Sprintf("    Ports: %s\n", strings.Join(info.Ports, ", ")))
		}
	}

	return engine.NewResultWithData(sb.String(), status), nil
}

func (c *InfraCommands) logs(ctx context.Context, params map[string]any) (engine.Result, error) {
	p := engine.Params(params)
	service := p.String("service", "")

	logs, err := c.composeSvc.Logs(ctx, service)
	if err != nil {
		return engine.ErrorResult(err), err
	}

	return engine.NewResult(logs), nil
}

func (c *InfraCommands) clean(ctx context.Context, params map[string]any) (engine.Result, error) {
	if err := c.composeSvc.Stop(ctx); err != nil {
		return engine.ErrorResult(err), err
	}
	return engine.NewResult("Infrastructure cleaned (containers, volumes, networks removed)"), nil
}
