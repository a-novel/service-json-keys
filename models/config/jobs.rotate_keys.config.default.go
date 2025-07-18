package config

import (
	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
)

const (
	JobsRotateKeysName = "service-json-keys-job-rotate-keys"
)

var JobRotateKeysPresetDefault = JobRotateKeys[*otelpresets.SentryOtelConfig, postgres.Config]{
	App: Main{
		Name:      config.LoadEnv(getEnv("APP_NAME"), JobsRotateKeysName, config.StringParser),
		MasterKey: getEnv("APP_MASTER_KEY"),
	},
	JWKS: JWKSPresetDefault,

	Otel: &otelpresets.SentryOtelConfig{
		DSN:          getEnv("SENTRY_DSN"),
		ServerName:   config.LoadEnv(getEnv("APP_NAME"), AppName, config.StringParser),
		Release:      getEnv("SENTRY_RELEASE"),
		Environment:  lo.CoalesceOrEmpty(getEnv("SENTRY_ENVIRONMENT"), getEnv("ENV")),
		FlushTimeout: config.LoadEnv(getEnv("SENTRY_FLUSH_TIMEOUT"), SentryFlushTimeout, config.DurationParser),
		Debug: config.LoadEnv(
			lo.CoalesceOrEmpty(getEnv("SENTRY_DEBUG"), getEnv("DEBUG")), false, config.BoolParser,
		),
	},
	Postgres: PostgresPresetDefault,
}
