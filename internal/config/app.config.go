package config

import (
	"time"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/servicetoken"
)

// Main holds the core application identity and secrets.
type Main struct {
	// Name is the application name, as it appears in logs and tracing.
	Name string
	// MasterKey is the hex-encoded 32-byte secret that encrypts private JSON Web Keys in the
	// database.
	MasterKey string
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time
}

// App is the complete configuration the servers read at startup.
type App struct {
	// App holds the core application identity and secrets.
	App Main
	// Grpc holds the gRPC server configuration.
	Grpc grpcf.ServerConfig
	// Rest holds the REST server configuration.
	Rest httpf.ServerConfig
	// ServiceToken names the service tokens the gRPC server accepts. Without an issuer, the server
	// refuses to start unless its IssuerOptional is set, which only a local or test stack does.
	ServiceToken servicetoken.Config
	// APIKeys holds the system API keys of each purpose, under its name: one or more Ed25519 public
	// keys in PEM, the active one first. Both servers serve them, and a purpose may hold none.
	APIKeys map[string]string

	// Otel configures the OpenTelemetry exporter for traces and metrics.
	Otel otel.Config
	// Logger is the application logger the REST handlers report their errors to.
	Logger logging.Log
	// GrpcLogger configures the gRPC request logging middleware.
	GrpcLogger logging.RPCConfig
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HTTPConfig
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config
}
