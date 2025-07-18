package config

import (
	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/models"
)

type JobRotateKeys[Otel otel.Config, Pg postgres.Config] struct {
	App  Main                      `json:"app"  yaml:"app"`
	JWKS map[models.KeyUsage]*JWKS `json:"jwks" yaml:"jwks"`

	Otel     Otel `json:"otel"     yaml:"otel"`
	Postgres Pg   `json:"postgres" yaml:"postgres"`
}
