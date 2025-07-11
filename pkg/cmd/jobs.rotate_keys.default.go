package cmdpkg

import (
	"os"

	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/migrations"
	"github.com/a-novel/service-json-keys/models"
)

const (
	JobsRotateKeysName = "job-rotate-keys"
)

var JobRotateKeysDefault = JobRotateKeysConfig[*otelpresets.SentryOtelConfig, postgres.Config]{
	App: AppAppConfig{
		Name:      config.LoadEnv(os.Getenv("APP_NAME"), JobsRotateKeysName, config.StringParser),
		MasterKey: os.Getenv("APP_MASTER_KEY"),
	},
	JWKS: models.DefaultJWKSConfig,

	Otel: &otelpresets.SentryOtelConfig{
		DSN:          os.Getenv("SENTRY_DSN"),
		ServerName:   config.LoadEnv(os.Getenv("APP_NAME"), AppName, config.StringParser),
		Release:      os.Getenv("SENTRY_RELEASE"),
		Environment:  lo.CoalesceOrEmpty(os.Getenv("SENTRY_ENVIRONMENT"), os.Getenv("ENV")),
		FlushTimeout: config.LoadEnv(os.Getenv("SENTRY_FLUSH_TIMEOUT"), SentryFlushTimeout, config.DurationParser),
		Debug: config.LoadEnv(
			lo.CoalesceOrEmpty(os.Getenv("SENTRY_DEBUG"), os.Getenv("DEBUG")), false, config.BoolParser,
		),
	},
	Postgres: postgrespresets.DefaultConfig{
		DSN:        os.Getenv("POSTGRES_DSN"),
		Migrations: migrations.Migrations,
	},
}
