package config

import (
	"time"

	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// RestCors holds CORS configuration for the REST server.
type RestCors struct {
	AllowedOrigins   []string `json:"allowedOrigins"   yaml:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"   yaml:"allowedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge"           yaml:"maxAge"`
}

// Main holds the core application identity and secrets.
type Main struct {
	// Name is the application name, as it appears in logs and tracing.
	Name string `json:"name" yaml:"name"`
	// MasterKey is a secure, 32-byte random secret used to encrypt private JSON Web Keys
	// in the database.
	MasterKey string `json:"masterKey" yaml:"masterKey"`
}

// Grpc holds the gRPC server configuration.
type Grpc struct {
	// Port is the port on which the gRPC server listens for incoming requests.
	Port int `json:"port" yaml:"port"`
	// Ping configures the refresh interval for the gRPC server internal healthcheck.
	Ping time.Duration `json:"ping" yaml:"ping"`
}

// RestTimeouts holds timeout configuration for the REST server.
type RestTimeouts struct {
	Read       time.Duration `json:"read"       yaml:"read"`
	ReadHeader time.Duration `json:"readHeader" yaml:"readHeader"`
	Write      time.Duration `json:"write"      yaml:"write"`
	Idle       time.Duration `json:"idle"       yaml:"idle"`
	Request    time.Duration `json:"request"    yaml:"request"`
}

// Rest holds the REST server configuration.
type Rest struct {
	// Port is the port on which the REST server listens for incoming requests.
	Port int `json:"port" yaml:"port"`
	// Timeouts holds the REST server timeout configuration.
	Timeouts RestTimeouts `json:"timeouts" yaml:"timeouts"`
	// MaxRequestSize is the maximum size of an incoming request body.
	MaxRequestSize int64 `json:"maxRequestSize" yaml:"maxRequestSize"`
	// Cors holds the CORS configuration.
	Cors RestCors `json:"cors" yaml:"cors"`
}

// App aggregates the configuration needed to run the gRPC and REST servers.
type App struct {
	// App holds the core application identity and secrets.
	App Main `json:"app" yaml:"app"`
	// Grpc holds the gRPC server configuration.
	Grpc Grpc `json:"grpc" yaml:"grpc"`
	// Rest holds the REST server configuration.
	Rest Rest `json:"rest" yaml:"rest"`

	// Otel configures the OpenTelemetry exporter for traces and metrics.
	Otel otel.Config `json:"otel" yaml:"otel"`
	// Logger is the base application logger used for general output.
	Logger logging.Log `json:"logger" yaml:"logger"`
	// GrpcLogger configures the gRPC request logging middleware.
	GrpcLogger logging.RpcConfig `json:"grpclogger" yaml:"grpclogger"`
	// RestLogger configures the REST request logging middleware.
	RestLogger logging.HttpConfig `json:"restLogger" yaml:"restLogger"`
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config `json:"postgres" yaml:"postgres"`
}
