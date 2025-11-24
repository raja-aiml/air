// Package air provides re-exports of all foundation and testinfra packages for building AI agents and MCP servers.
// This single package exports everything needed from air in one place.
package air

import (
	"context"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raja-aiml/air/internal/commands"
	"github.com/raja-aiml/air/internal/engine"
	"github.com/raja-aiml/air/internal/foundation/auth"
	"github.com/raja-aiml/air/internal/foundation/compose"
	"github.com/raja-aiml/air/internal/foundation/config"
	db "github.com/raja-aiml/air/internal/foundation/database"
	"github.com/raja-aiml/air/internal/foundation/errors"
	ghpub "github.com/raja-aiml/air/internal/foundation/github"
	"github.com/raja-aiml/air/internal/foundation/httpclient"
	"github.com/raja-aiml/air/internal/foundation/logging"
	"github.com/raja-aiml/air/internal/foundation/observability/metrics"
	telemetry "github.com/raja-aiml/air/internal/foundation/observability/tracing"
	"github.com/raja-aiml/air/internal/mcp"
	"github.com/raja-aiml/air/internal/nlp"
	"github.com/raja-aiml/air/internal/testinfra/containers"
	"github.com/raja-aiml/air/internal/testinfra/tests"
	"github.com/raja-aiml/air/internal/testinfra/verification"
)

// ============================================================================
// COMPOSE - Docker Compose SDK Management
// ============================================================================

type (
	ComposeService       = compose.Service
	ComposeServiceStatus = compose.ServiceStatus
	ComposeServiceInfo   = compose.ServiceInfo
	ComposeConfig        = compose.Config
)

func NewComposeService(cfg ComposeConfig) (*ComposeService, error) {
	return compose.New(cfg)
}

// ============================================================================
// HTTP - HTTP Client Utilities
// ============================================================================

type HTTPClient = httpclient.Client

var (
	NewHTTPClient     = httpclient.New
	DefaultHTTPClient = httpclient.Default
)

// ============================================================================
// CONFIG - Configuration Loading
// ============================================================================

type (
	ServerConfig   = config.ServerConfig
	BackfillConfig = config.BackfillConfig
	JWTGenConfig   = config.JWTGenConfig
)

var (
	LoadServerConfig   = config.LoadServerConfig
	LoadBackfillConfig = config.LoadBackfillConfig
	LoadJWTGenConfig   = config.LoadJWTGenConfig
	ParseLogLevel      = config.ParseLogLevel
	ParseInt           = config.ParseInt
)

// ============================================================================
// DATABASE - Connection Pooling & Migrations
// ============================================================================

func NewDatabasePool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	return db.NewPool(ctx, url)
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	return db.RunMigrations(ctx, pool)
}

func PingDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	return db.Ping(ctx, pool)
}

// ============================================================================
// AUTH - JWT Token Management
// ============================================================================

type TokenClaims = auth.TokenClaims

func GenerateJWTToken(claims TokenClaims, secret string) (string, error) {
	return auth.GenerateToken(claims, secret)
}

// ============================================================================
// ERRORS - Structured Error Handling
// ============================================================================

type AppError = errors.AppError

const (
	ErrCodeInvalidEnvelope     = errors.ErrCodeInvalidEnvelope
	ErrCodeInvalidEvent        = errors.ErrCodeInvalidEvent
	ErrCodeUnknownEvent        = errors.ErrCodeUnknownEvent
	ErrCodeUnimplemented       = errors.ErrCodeUnimplemented
	ErrCodeInvalidPayload      = errors.ErrCodeInvalidPayload
	ErrCodePayloadTooLarge     = errors.ErrCodePayloadTooLarge
	ErrCodeAuthFailed          = errors.ErrCodeAuthFailed
	ErrCodeInvalidToken        = errors.ErrCodeInvalidToken
	ErrCodeTokenExpired        = errors.ErrCodeTokenExpired
	ErrCodeTokenMissing        = errors.ErrCodeTokenMissing
	ErrCodeUnauthorized        = errors.ErrCodeUnauthorized
	ErrCodeDatabaseUnavailable = errors.ErrCodeDatabaseUnavailable
	ErrCodeDatabaseQuery       = errors.ErrCodeDatabaseQuery
	ErrCodeDatabaseConstraint  = errors.ErrCodeDatabaseConstraint
	ErrCodeNotFound            = errors.ErrCodeNotFound
	ErrCodeRateLimited         = errors.ErrCodeRateLimited
	ErrCodeInternal            = errors.ErrCodeInternal
)

var (
	NewError            = errors.New
	WrapError           = errors.Wrap
	IsErrorCode         = errors.Is
	InvalidEnvelope     = errors.InvalidEnvelope
	InvalidEvent        = errors.InvalidEvent
	UnknownEvent        = errors.UnknownEvent
	Unimplemented       = errors.Unimplemented
	InvalidPayload      = errors.InvalidPayload
	PayloadTooLarge     = errors.PayloadTooLarge
	AuthFailed          = errors.AuthFailed
	InvalidToken        = errors.InvalidToken
	TokenMissing        = errors.TokenMissing
	Unauthorized        = errors.Unauthorized
	DatabaseUnavailable = errors.DatabaseUnavailable
	DatabaseQuery       = errors.DatabaseQuery
	NotFound            = errors.NotFound
	RateLimited         = errors.RateLimited
	Internal            = errors.Internal
)

// ============================================================================
// LOGGING - Structured Logging
// ============================================================================

var InitLogger = logging.InitLogger

// ============================================================================
// TRACING - OpenTelemetry Tracing
// ============================================================================

type (
	DBTracer  = telemetry.DBTracer
	Span      = trace.Span
	Attribute = attribute.KeyValue
)

var (
	InitTracer        = telemetry.InitTracer
	GetTracer         = telemetry.Tracer
	GetTraceID        = telemetry.GetTraceID
	AddSpanAttributes = telemetry.AddSpanAttributes
	LogInfo           = telemetry.LogInfo
	LogDebug          = telemetry.LogDebug
	LogWarn           = telemetry.LogWarn
	LogError          = telemetry.LogError
	LogEvent          = telemetry.LogEvent
	WithCorrelationID = telemetry.WithCorrelationID
	GetCorrelationID  = telemetry.GetCorrelationID
	WithRequestID     = telemetry.WithRequestID
	GetRequestID      = telemetry.GetRequestID
	WithUserID        = telemetry.WithUserID
	GetUserID         = telemetry.GetUserID
	WithSessionID     = telemetry.WithSessionID
	GetSessionID      = telemetry.GetSessionID
	NewCorrelationID  = telemetry.NewCorrelationID
	EnrichContext     = telemetry.EnrichContext
)

func NewDBTracer() *DBTracer {
	return telemetry.NewDBTracer()
}

// ============================================================================
// ENGINE - Command registry and command types (re-exported from internal/engine)
// ============================================================================

type (
	Registry  = engine.Registry
	Command   = engine.Command
	Result    = engine.Result
	Params    = engine.Params
	Parameter = engine.Parameter
)

var (
	NewRegistry = engine.NewRegistry
)

// ============================================================================
// COMMANDS - Command groups builders (re-exported from internal/commands)
// ============================================================================

type (
	InfraCommands = commands.InfraCommands
	DBCommands    = commands.DBCommands
)

var (
	NewInfraCommands = commands.NewInfraCommands
	NewDBCommands    = commands.NewDBCommands
	NewObsCommands   = commands.NewObsCommands
	NewLintCommands  = commands.NewLintCommands
)

// ============================================================================
// MCP - MCP server wrapper (re-exported)
// ============================================================================

type (
	MCPServer = mcp.Server
	MCPConfig = mcp.Config
)

var (
	NewMCPServer     = mcp.NewServer
	DefaultMCPConfig = mcp.DefaultConfig
)

// ============================================================================
// NLP - Parser (re-exported)
// ============================================================================

type (
	Parser       = nlp.Parser
	ParserConfig = nlp.ParserConfig
)

var (
	NewParser           = nlp.NewParser
	DefaultParserConfig = nlp.DefaultParserConfig
)

// ============================================================================
// TESTS - testinfra helpers
// ============================================================================

type (
	TestingT = tests.TestingT
)

var (
	NewManualTester = tests.NewManualTester
)

var (
	VerifyTracesPropagation = tests.VerifyTracesPropagation
	VerifyMetricsCollection = tests.VerifyMetricsCollection
)

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return telemetry.Tracer().Start(ctx, name, opts...)
}

// ============================================================================
// METRICS - Application Metrics
// ============================================================================

type (
	Metrics    = metrics.Metrics
	Stats      = metrics.Stats
	EventStats = metrics.EventStats
)

var (
	GetMetrics     = metrics.GetMetrics
	IncWebSocket   = metrics.IncWS
	DecWebSocket   = metrics.DecWS
	MetricsHandler = metrics.MetricsHandler
)

func RecordEvent(eventName string, duration time.Duration) {
	GetMetrics().WSEventProcessed(eventName, duration)
}

func RecordError(eventName string) {
	GetMetrics().WSEventError(eventName)
}

func GetCurrentStats() Stats {
	return GetMetrics().GetStats()
}

// ============================================================================
// TESTINFRA - Testing Infrastructure
// ============================================================================

type (
	Infrastructure = containers.Infrastructure
	TestConfig     = containers.Config
	Report         = containers.Report
)

var (
	DefaultTestConfig         = containers.DefaultConfig
	StartWithCompose          = containers.StartWithCompose
	StartInfrastructure       = containers.StartInfrastructure
	StartServerInBackground   = containers.StartServerInBackground
	VerifyContainerHealth     = containers.VerifyContainerHealth
	StartApplicationServer    = containers.StartApplicationServer
	CleanupInfrastructure     = containers.CleanupInfrastructure
	NewReport                 = containers.NewReport
	WaitForPostgres           = containers.WaitForPostgres
	WaitForJaeger             = containers.WaitForJaeger
	WaitForPrometheus         = containers.WaitForPrometheus
	WaitForHTTP               = containers.WaitForHTTP
	WaitForSchema             = containers.WaitForSchema
	VerifyPostgresHealth      = containers.VerifyPostgresHealth
	VerifyJaegerHealth        = containers.VerifyJaegerHealth
	VerifyPrometheusHealth    = containers.VerifyPrometheusHealth
	VerifyOtelCollectorHealth = containers.VerifyOtelCollectorHealth
	ApplyMigrations           = containers.ApplyMigrations
)

// Helper: Start test infrastructure with cleanup
func StartTestInfrastructure(ctx context.Context) (*Infrastructure, func(), error) {
	cfg := DefaultTestConfig()

	infra, err := StartWithCompose(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		CleanupInfrastructure(infra)
	}

	return infra, cleanup, nil
}

// Helper: Wait for all services with timeout
func WaitForAllServices(ctx context.Context, infra *Infrastructure, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	if err := WaitForPostgres(ctx, infra.PostgresURL); err != nil {
		return err
	}

	if err := WaitForJaeger(ctx, infra.JaegerURL); err != nil {
		return err
	}

	if err := WaitForPrometheus(ctx, infra.PrometheusURL); err != nil {
		return err
	}

	return nil
}

// Helper: Get container logs
func GetContainerLogs(ctx context.Context, infra *Infrastructure, containerType string) (io.ReadCloser, error) {
	return infra.GetContainerLogs(ctx, containerType)
}

// ============================================================================
// VERIFICATION - Observability Verification
// ============================================================================

var RunVerification = verification.Run

func VerifyObservability(ctx context.Context) error {
	cfg := DefaultTestConfig()
	return RunVerification(ctx, cfg, false)
}

func VerifyObservabilityJSON(ctx context.Context) error {
	cfg := DefaultTestConfig()
	return RunVerification(ctx, cfg, true)
}

// ============================================================================
// GITHUB - Repository Publishing
// ============================================================================

type (
	GitHubPublisher  = ghpub.Publisher
	RepositoryConfig = ghpub.RepositoryConfig
	ReleaseConfig    = ghpub.ReleaseConfig
	PublishOptions   = ghpub.PublishOptions
)

var (
	NewGitHubPublisher = ghpub.NewPublisher
	PublishToGitHub    = ghpub.Publish
)
