package config

import (
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// JobRotateKeys is the configuration for the key-rotation background job.
type JobRotateKeys struct {
	// App holds the core application identity and secrets.
	App Main `json:"app" yaml:"app"`
	// Jwk holds the signing key configuration for each registered usage, keyed by usage name.
	Jwk map[string]*Jwk `json:"jwk" yaml:"jwk"`

	// Otel configures the OpenTelemetry exporter for traces and metrics.
	Otel otel.Config `json:"otel" yaml:"otel"`
	// Postgres configures the PostgreSQL connection.
	Postgres postgres.Config `json:"postgres" yaml:"postgres"`
}
