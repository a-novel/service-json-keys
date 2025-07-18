package config

import (
	"time"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/models"
)

type Main struct {
	Name      string `json:"name"      yaml:"name"`
	MasterKey string `json:"masterKey" yaml:"masterKey"`
}

type APITimeouts struct {
	Read       time.Duration `json:"read"       yaml:"read"`
	ReadHeader time.Duration `json:"readHeader" yaml:"readHeader"`
	Write      time.Duration `json:"write"      yaml:"write"`
	Idle       time.Duration `json:"idle"       yaml:"idle"`
	Request    time.Duration `json:"request"    yaml:"request"`
}

type Cors struct {
	AllowedOrigins   []string `json:"allowedOrigins"   yaml:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"   yaml:"allowedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge"           yaml:"maxAge"`
}

type API struct {
	Port           int         `json:"port"           yaml:"port"`
	Timeouts       APITimeouts `json:"timeouts"       yaml:"timeouts"`
	MaxRequestSize int64       `json:"maxRequestSize" yaml:"maxRequestSize"`
	Cors           Cors        `json:"cors"           yaml:"cors"`
}

type App[Otel otel.Config, Pg postgres.Config] struct {
	App  Main                      `json:"app"  yaml:"app"`
	API  API                       `json:"api"  yaml:"api"`
	JWKS map[models.KeyUsage]*JWKS `json:"jwks" yaml:"jwks"`

	Otel     Otel `json:"otel"     yaml:"otel"`
	Postgres Pg   `json:"postgres" yaml:"postgres"`
}
