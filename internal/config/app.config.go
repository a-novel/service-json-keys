package config

import (
	"time"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// Main holds the core application identity and secrets.
type Main struct {
	// Name is the application name, as it appears in logs and tracing.
	Name string `json:"name" yaml:"name"`
	// MasterKey is a secure, 32-byte random secret used to encrypt private JSON Web Keys
	// in the database.
	MasterKey string `json:"masterKey" yaml:"masterKey"`
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time `json:"downtimeStart" yaml:"downtimeStart"`
}

// App aggregates the configuration needed to run the gRPC and REST servers.
type App struct {
	// App holds the core application identity and secrets.
	App Main `json:"app" yaml:"app"`
	// Grpc holds the gRPC server configuration.
	Grpc grpcf.ServerConfig `json:"grpc" yaml:"grpc"`
	// Rest holds the REST server configuration.
	Rest httpf.ServerConfig `json:"rest" yaml:"rest"`

	// Otel configures the OpenTelemetry exporter for traces and metrics.
	Otel otel.Config `json:"otel" yaml:"otel"`
	// Logger is the base application logger used for general output.
	Logger logging.Log `json:"logger" yaml:"logger"`
	// GrpcLogger configures the gRPC request logging middleware.
	GrpcLogger logging.RPCConfig `json:"grpclogger" yaml:"grpclogger"`
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HTTPConfig `json:"restLogger" yaml:"restLogger"`
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config `json:"postgres" yaml:"postgres"`
}
