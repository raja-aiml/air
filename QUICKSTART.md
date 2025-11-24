# Quick Start Guide - air

Get started with **air** in 5 minutes.

## Installation

### Option 1: As a Go Library

```bash
# Add to your project
go get github.com/raja-aiml/air@latest
```

### Option 2: CLI Tools

```bash
# Install CLI tools
go install github.com/raja-aiml/air/cmd/air-dev@latest
go install github.com/raja-aiml/air/cmd/air-verify@latest
```

## Your First AI Agent with air

### 1. Create a New Project

```bash
mkdir my-ai-agent
cd my-ai-agent
go mod init my-ai-agent
go get github.com/raja-aiml/air@latest
```

### 2. Create `main.go`

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "encoding/json"
    
    "github.com/raja-aiml/air"
)

func main() {
    ctx := context.Background()
    
    // 1. Initialize observability
    shutdown := air.InitTracer("my-ai-agent")
    defer shutdown()
    
    // 2. Setup logging
    air.InitLogger("development")
    
    // 3. Connect to database (optional)
    // pool := air.NewDatabasePool(ctx, "postgres://...")
    // defer pool.Close()
    
    // 4. Create MCP server endpoint
    http.HandleFunc("/mcp/tools/invoke", handleToolInvocation)
    
    fmt.Println("🚀 AI Agent starting on http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}

func handleToolInvocation(w http.ResponseWriter, r *http.Request) {
    // Trace the entire tool invocation
    ctx, span := air.StartSpan(r.Context(), "mcp-tool-invoke")
    defer span.End()
    
    // Parse request
    var req struct {
        Tool   string                 `json:"tool"`
        Params map[string]interface{} `json:"params"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Add AI-specific tracing attributes
    air.AddSpanAttributes(span,
        air.Attribute{Key: "mcp.tool", Value: air.AttributeValue{StringValue: req.Tool}},
    )
    
    air.LogInfo(ctx, "Executing tool", "tool", req.Tool)
    
    // Your AI logic here...
    result := map[string]interface{}{
        "tool":   req.Tool,
        "status": "success",
        "result": "Tool executed successfully",
    }
    
    json.NewEncoder(w).Encode(result)
}
```

### 3. Start Infrastructure

```bash
# Using CLI tool
air-dev up

# Or using docker compose directly
cd /path/to/air
docker compose -f config/docker/compose-template.yml up -d
```

### 4. Run Your Agent

```bash
go run main.go
```

### 5. Test It

```bash
# Test the MCP endpoint
curl -X POST http://localhost:8080/mcp/tools/invoke \
  -H "Content-Type: application/json" \
  -d '{"tool": "search", "params": {"query": "test"}}'
```

### 6. View Observability

```bash
# Open Jaeger (traces)
open http://localhost:16686

# Open Prometheus (metrics)
open http://localhost:9090

# Or use CLI
air-verify
```

## Testing Your Agent

### Create `main_test.go`

```go
package main

import (
    "context"
    "testing"
    
    "github.com/raja-aiml/air"
)

func TestAgentInfrastructure(t *testing.T) {
    ctx := context.Background()
    
    // Start full test infrastructure
    infra, cleanup, err := air.StartTestInfrastructure(ctx)
    if err != nil {
        t.Fatalf("Failed to start infrastructure: %v", err)
    }
    defer cleanup()
    
    // Now you have:
    // - PostgreSQL with pgvector
    // - Jaeger for traces
    // - Prometheus for metrics
    // - OTEL Collector
    
    t.Logf("Jaeger UI: %s", infra.JaegerURL)
    t.Logf("Prometheus UI: %s", infra.PrometheusURL)
    
    // Test your agent logic...
}
```

### Run Tests

```bash
go test -v
```

## What You Get

✅ **Distributed Tracing** - Every request traced in Jaeger  
✅ **Metrics** - Application metrics in Prometheus  
✅ **Structured Logs** - Correlated logs with trace IDs  
✅ **Database** - PostgreSQL with pgvector for embeddings  
✅ **Testing** - Full infrastructure for integration tests  

## Next Steps

- 📖 Read the [Full Documentation](README.md)
- 🔧 Explore [Examples](examples/)
- 🤝 [Contributing Guide](CONTRIBUTING.md)
- 📦 Check [Publishing Guide](PUBLISHING.md)

## Common Commands

```bash
# Start infrastructure
air-dev up

# Stop infrastructure  
air-dev down

# Check status
air-dev status

# Verify observability
air-verify

# View logs
air-dev logs postgres
```

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 8080
lsof -ti:8080

# Kill the process
lsof -ti:8080 | xargs kill -9
```

### Infrastructure Not Starting

```bash
# Check Docker
docker ps

# Restart infrastructure
air-dev down
air-dev up
```

### Traces Not Appearing

```bash
# Verify OTEL Collector is running
curl http://localhost:4318/v1/traces

# Check Jaeger
curl http://localhost:16686
```

## Getting Help

- 📖 [Full README](README.md)
- 🐛 [Report Issues](https://github.com/raja-aiml/air/issues)
- 💬 [Discussions](https://github.com/raja-aiml/air/discussions)

---

**You're now ready to build production-ready AI agents with full observability! 🚀**
