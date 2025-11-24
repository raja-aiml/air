<div align="center">

# 🌊 air

**AI Runtime Infrastructure**

*Build production-ready AI agents and MCP servers in Go with batteries-included observability, testing, and infrastructure*

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Required-2496ED?style=flat&logo=docker)](https://www.docker.com/)

[Features](#-features) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [Examples](#-examples) • [Contributing](#-contributing)

</div>

---

## 🎯 What is air?

**air** is a comprehensive, production-ready foundation for building observable **AI agents and MCP (Model Context Protocol) servers** in Go. It provides everything you need to ship intelligent, context-aware AI services:

- 🔭 **Full Observability Stack** - Track agent reasoning, tool calls, and context propagation with OpenTelemetry
- 🗄️ **Database Management** - Store agent state, conversation history, and embeddings
- 🧪 **Testing Infrastructure** - Test AI workflows, tool integrations, and context handling
- 🐳 **Docker Compose Integration** - Managed infrastructure for vector databases and observability
- 🛠️ **Developer Tools** - Debug agent behavior, trace decisions, monitor performance

Stop building infrastructure. Start building intelligent agents.

---

## ✨ Features

### 📊 AI-Native Observability

```go
import "github.com/raja-aiml/air"

func main() {
    // One line to enable full observability for AI agents
    shutdown := air.InitTracer("my-ai-agent")
    defer shutdown()
    
    // Track agent reasoning, tool calls, and context flow
    ctx, span := air.StartSpan(ctx, "agent-reasoning")
    air.AddSpanAttributes(span,
        air.Attribute("agent.tool", "search"),
        air.Attribute("agent.context_tokens", 1500),
    )
    defer span.End()
    
    air.LogInfo(ctx, "Agent processing query")
    air.RecordEvent("tool_invocation", duration)
}
```

**Included:**
- ✅ Trace AI agent decisions and tool calls
- ✅ Monitor MCP server tool invocations
- ✅ Track context usage and token consumption
- ✅ Correlate agent actions across distributed systems
- ✅ Debug agent behavior with structured logs

### 🗄️ AI-Ready Database Foundation

```go
// Production-ready connection pooling for AI workloads
pool := air.NewDatabasePool(ctx, databaseURL)
defer pool.Close()

// Automatic migrations (including pgvector for embeddings)
air.RunMigrations(ctx, pool)

// Store agent state and conversation history
air.PingDatabase(ctx, pool)
```

**Features:**
- ✅ PostgreSQL with pgvector support for embeddings
- ✅ Store agent state, memory, and context
- ✅ Conversation history and session management
- ✅ Distributed tracing for all database operations
- ✅ Perfect for RAG (Retrieval Augmented Generation)

### 🧪 Test AI Workflows End-to-End

```go
func TestAIAgent(t *testing.T) {
    // Start full infrastructure with one call
    infra, cleanup, err := air.StartTestInfrastructure(ctx)
    defer cleanup()
    
    // Test agent behavior with full observability
    // Verify tool calls, context handling, and responses
    // All traces available in Jaeger for debugging
}
```

**Included:**
- ✅ Test MCP server implementations
- ✅ Verify agent tool invocations
- ✅ Validate context propagation
- ✅ Debug with full trace visibility
- ✅ Mock external AI services

### 🛠️ Developer Experience

```bash
# Start all infrastructure services
air dev up

# Verify observability pipeline
air verify

# Check service status
air dev status

# View service logs
air dev logs postgres
```

**CLI Tools:**
- ✅ `air dev` - Infrastructure management
- ✅ `air verify` - Observability verification
- ✅ Rich terminal output with progress indicators
- ✅ Automatic health checks

---

## 🚀 Quick Start

### Installation

```bash
# Add air to your project
go get github.com/raja-aiml/air

# Or use as a module
git clone https://github.com/raja-aiml/air.git
```

### Project Setup

**Option 1: Use the CLI**

```bash
# Quick start with CLI
air dev up
air verify
```

**Option 2: Use Go Package Directly for AI Agents**

```go
package main

import "github.com/raja-aiml/air"

func main() {
    // Initialize observability for AI workloads
    shutdown := air.InitTracer("my-ai-agent")
    defer shutdown()
    
    // Initialize logger
    air.InitLogger("production")
    
    // Connect to database (with pgvector for embeddings)
    pool := air.NewDatabasePool(ctx, databaseURL)
    defer pool.Close()
    
    // Start your MCP server or AI agent...
}
```

### Start Infrastructure

```bash
# Option 1: Using CLI
air dev up

# Option 2: Using Make
make dev-up

# Option 3: Using Docker Compose
cd air/config/docker && docker compose up -d
```

### Verify Everything Works

```bash
air verify
```

You'll see:
- ✅ Infrastructure health checks
- ✅ Observability pipeline verification
- ✅ Trace propagation tests
- ✅ Metrics collection validation

---

## 📚 Documentation

### Available Make Targets

```bash
# Infrastructure
make infra-up          # Start postgres + observability stack
make infra-down        # Stop all services
make infra-logs        # View service logs
make infra-clean       # Remove volumes and networks

# Database
make db-migrate        # Run migrations
make db-seed           # Load sample data
make db-reset          # Reset database (destructive)
make db-shell          # Open psql shell

# Observability
make obs-verify        # Health check all services
make obs-jaeger        # Open Jaeger UI (http://localhost:16686)
make obs-prometheus    # Open Prometheus UI (http://localhost:9090)

# Development
make dev-up            # Start full dev environment
make dev-down          # Stop dev environment

# Testing
make test-unit         # Run unit tests
make test-integration  # Run integration tests
make test-all          # Run all tests
make test-coverage     # Generate coverage report

# CI/CD
make ci-check          # Run all CI checks
make ci-build          # Build all binaries
```

### Project Structure

```
air/
├── pkg/                    # 📦 Public API (import this)
│   └── airuntime.go       # Single unified package export
│
├── internal/               # 🔒 Private implementation
│   ├── foundation/
│   │   ├── compose/       # Docker Compose SDK
│   │   ├── config/        # Configuration management
│   │   ├── database/      # pgx connection pooling
│   │   ├── auth/          # JWT utilities
│   │   ├── errors/        # Structured error handling
│   │   ├── logging/       # Structured logging (zerolog)
│   │   └── observability/
│   │       ├── tracing/   # OpenTelemetry tracing
│   │       ├── metrics/   # Prometheus metrics
│   │       └── logging/   # Log correlation
│   └── testinfra/
│       ├── containers/    # Testcontainers orchestration
│       ├── tests/         # Test helpers
│       └── verification/  # Health checks
│
├── cmd/
│   ├── dev/               # Infrastructure management CLI
│   └── verify/            # Observability verification CLI
│
├── config/
│   ├── docker/
│   │   └── compose-template.yml
│   ├── database/
│   │   └── 001_init.sql
│   └── observability/
│       ├── otel-collector-config.yaml
│       ├── prometheus-config.yaml
│       └── fluent-bit.conf
```

---

## 💡 Examples

### Complete MCP Server Example

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    
    "github.com/raja-aiml/air"
)

func main() {
    ctx := context.Background()
    
    // 1. Initialize observability for MCP server
    shutdown := air.InitTracer("my-mcp-server")
    defer shutdown()
    
    // 2. Setup structured logging
    air.InitLogger("production")
    
    // 3. Connect to database (for context/state storage)
    pool := air.NewDatabasePool(ctx, 
        "postgres://postgres:postgres@localhost:5432/mcp_db")
    defer pool.Close()
    
    // 4. Run migrations (includes pgvector)
    if err := air.RunMigrations(ctx, pool); err != nil {
        panic(err)
    }
    
    // 5. MCP server endpoint - tool invocation
    http.HandleFunc("/mcp/tools/invoke", func(w http.ResponseWriter, r *http.Request) {
        // Trace the entire tool invocation
        ctx, span := air.StartSpan(r.Context(), "mcp-tool-invoke")
        defer span.End()
        
        // Parse tool request
        var req struct {
            Tool   string                 `json:"tool"`
            Params map[string]interface{} `json:"params"`
        }
        json.NewDecoder(r.Body).Decode(&req)
        
        // Add AI-specific attributes to trace
        air.AddSpanAttributes(span,
            air.Attribute("mcp.tool", req.Tool),
            air.Attribute("mcp.params", req.Params),
        )
        
        air.LogInfo(ctx, "Executing MCP tool", "tool", req.Tool)
        
        // Execute tool (your AI logic here)
        result := executeTool(ctx, pool, req.Tool, req.Params)
        
        // Record metrics
        air.RecordEvent("mcp_tool_invocation", duration)
        
        json.NewEncoder(w).Encode(result)
    })
    
    // 6. Start MCP server
    air.LogInfo(ctx, "MCP Server starting on :8080")
    http.ListenAndServe(":8080", nil)
}

func executeTool(ctx context.Context, pool *pgxpool.Pool, 
    tool string, params map[string]interface{}) interface{} {
    
    // Trace individual tool execution
    ctx, span := air.StartSpan(ctx, "tool-execution")
    air.AddSpanAttributes(span, air.Attribute("tool.name", tool))
    defer span.End()
    
    // Your AI agent logic here
    // - Query vector database
    // - Call external APIs
    // - Process with LLM
    
    return map[string]interface{}{"result": "success"}
}
```

### Testing AI Agent Example

```go
func TestAIAgent(t *testing.T) {
    ctx := context.Background()
    
    // Start infrastructure
    infra, cleanup, err := air.StartTestInfrastructure(ctx)
    require.NoError(t, err)
    defer cleanup()
    
    // Connect to test database (with pgvector)
    pool := air.NewDatabasePool(ctx, infra.PostgresURL)
    defer pool.Close()
    
    // Test AI agent workflows
    t.Run("agent tool invocation", func(t *testing.T) {
        // Simulate MCP tool call
        ctx, span := air.StartSpan(ctx, "test-tool-call")
        defer span.End()
        
        // Test agent behavior
        result := invokeMCPTool(ctx, pool, "search", params)
        assert.NotNil(t, result)
        
        // Verify traces captured
        // Check Jaeger for complete execution flow
    })
    
    t.Run("agent context handling", func(t *testing.T) {
        // Test context propagation
        // Verify memory/state persistence
    })
    
    // All traces and metrics available for debugging
    t.Logf("Jaeger UI: %s", infra.JaegerURL)
    t.Logf("View agent traces and tool calls")
}
```

### AI Agent Metrics Example

```go
// Record AI-specific metrics
air.RecordEvent("agent_tool_call", processingTime)
air.RecordEvent("llm_invocation", llmDuration)
air.RecordEvent("context_retrieval", retrievalTime)
air.RecordError("tool_execution_failed")

// Monitor agent performance
stats := air.GetCurrentStats()
fmt.Printf("Total tool calls: %d\n", stats.TotalEvents)
fmt.Printf("Agent error rate: %.2f%%\n", stats.ErrorRate)
fmt.Printf("Avg response time: %v\n", stats.AvgDuration)

// Track token usage, context size, embedding operations
// Monitor MCP server health and throughput
```

---

## 🏗️ Architecture

### Design Philosophy

**air** follows these principles for AI development:

1. **AI-First Architecture** - Built specifically for AI agents and MCP servers
2. **Observability-Driven** - Trace every agent decision, tool call, and context flow
3. **Production-Ready** - Battle-tested components for real AI workloads
4. **Developer Experience** - Debug agent behavior, not infrastructure
5. **Testability** - Test AI workflows end-to-end with full visibility

### Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Language** | Go 1.24+ | MCP servers, AI agents |
| **Database** | PostgreSQL + pgvector | Agent state, embeddings, RAG |
| **Tracing** | OpenTelemetry + Jaeger | Track agent decisions & tools |
| **Metrics** | Prometheus | Monitor agent performance |
| **Logging** | zerolog | Structured logs with context |
| **Testing** | Testcontainers | Test AI workflows |
| **Orchestration** | Docker Compose | Local AI infrastructure |

### AI Observability Pipeline

```
┌─────────────────┐
│   AI Agent      │  Your MCP server / AI agent
│  (air-enabled)  │  Tool calls, context, reasoning
└────────┬────────┘
         │ OTLP/gRPC (traces + metrics)
         ▼
┌─────────────────┐
│      OTEL       │  Collects:
│   Collector     │  - Agent decisions
└────────┬────────┘  - Tool invocations
         │           - Context usage
    ┌────┴────┐      - Token consumption
    │         │
    ▼         ▼
┌─────────┐ ┌───────────┐
│ Jaeger  │ │Prometheus │  Visualize:
│ (traces)│ │ (metrics) │  - Agent reasoning flow
└─────────┘ └───────────┘  - Performance bottlenecks
    │           │           - Error patterns
    └─────┬─────┘
          ▼
    📊 Debug AI Behavior
       - Why did the agent fail?
       - Which tools were called?
       - How much context was used?
```

---

## 🔧 Requirements

- **Go**: 1.24 or higher
- **Docker**: 20.10 or higher
- **Docker Compose**: v2.0 or higher
- **PostgreSQL**: 14+ (managed by air)

### Optional Tools

- `psql` - For database operations
- `golangci-lint` - For linting

---

## 🤝 Contributing

We welcome contributions! Here's how you can help:

1. 🐛 **Report Bugs** - Open an issue with reproduction steps
2. 💡 **Suggest Features** - Share your ideas for improvements
3. 📝 **Improve Docs** - Help us make documentation better
4. 🔧 **Submit PRs** - Fix bugs or add features

### Development Setup

```bash
# Clone repository
git clone https://github.com/raja-aiml/air.git
cd air

# Install dependencies
make deps

# Install development tools
make tools

# Run tests
make test-all

# Start infrastructure
make dev-up

# Verify everything works
air verify
```

---

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

Built with these excellent open-source projects:

- [OpenTelemetry](https://opentelemetry.io/) - Observability framework
- [Jaeger](https://www.jaegertracing.io/) - Distributed tracing
- [Prometheus](https://prometheus.io/) - Metrics and monitoring
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver
- [Testcontainers](https://www.testcontainers.org/) - Integration testing
- [zerolog](https://github.com/rs/zerolog) - Structured logging

---

## 🔗 Links

- 📖 [Documentation](https://github.com/raja-aiml/air/wiki)
- 🐛 [Issue Tracker](https://github.com/raja-aiml/air/issues)
- 💬 [Discussions](https://github.com/raja-aiml/air/discussions)
- 🎯 [Project Roadmap](https://github.com/raja-aiml/air/projects)

---

<div align="center">

**Made with ❤️ by developers, for developers**

[⬆ Back to Top](#-air)

</div>
