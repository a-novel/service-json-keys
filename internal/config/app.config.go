package config

import (
	"time"

	"github.com/a-novel/golib/logging"
	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

// Main application configuration.
type Main struct {
	// Name of the application, as it will appear in logs and tracing.
	Name string `json:"name" yaml:"name"`
	// MasterKey is a secure, 32-byte random secret used to encrypt private JSON keys
	// in the database.
	MasterKey string `json:"masterKey" yaml:"masterKey"`
}

// Grpc server configuration.
type Grpc struct {
	// Port on which the Grpc server will listen for incoming requests.
	Port int `json:"port" yaml:"port"`
	// Ping configures the refresh interval for the Grpc server internal healthcheck.
	Ping time.Duration `json:"ping" yaml:"ping"`
}

type App struct {
	App  Main `json:"app"  yaml:"app"`
	Grpc Grpc `json:"grpc" yaml:"grpc"`

	Otel     otel.Config       `json:"otel"     yaml:"otel"`
	Logger   logging.RpcConfig `json:"logger"   yaml:"logger"`
	Postgres postgres.Config   `json:"postgres" yaml:"postgres"`
}
