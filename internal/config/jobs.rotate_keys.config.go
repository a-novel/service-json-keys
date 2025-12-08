package config

import (
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

type JobRotateKeys struct {
	App Main `json:"app" yaml:"app"`
	// Jwk configuration for each supported usage.
	Jwk map[string]*Jwk `json:"jwk" yaml:"jwk"`

	Otel     otel.Config     `json:"otel"     yaml:"otel"`
	Postgres postgres.Config `json:"postgres" yaml:"postgres"`
}
