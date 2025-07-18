package config

import (
	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
)

func AppPresetTest(port int) App[*otelpresets.SentryOtelConfig, postgres.Config] {
	return App[*otelpresets.SentryOtelConfig, postgres.Config]{
		App: Main{
			Name:      config.LoadEnv(getEnv("APP_NAME"), AppName, config.StringParser),
			MasterKey: getEnv("APP_MASTER_KEY"),
		},
		API: API{
			Port:           port,
			MaxRequestSize: config.LoadEnv(getEnv("API_MAX_REQUEST_SIZE"), APIMaxRequestSize, config.Int64Parser),
			Timeouts: APITimeouts{
				Read: config.LoadEnv(getEnv("API_TIMEOUT_READ"), APITimeoutRead, config.DurationParser),
				ReadHeader: config.LoadEnv(
					getEnv("API_TIMEOUT_READ_HEADER"), APITimeoutReadHeader, config.DurationParser,
				),
				Write:   config.LoadEnv(getEnv("API_TIMEOUT_WRITE"), APITimeoutWrite, config.DurationParser),
				Idle:    config.LoadEnv(getEnv("API_TIMEOUT_IDLE"), APITimeoutIdle, config.DurationParser),
				Request: config.LoadEnv(getEnv("API_TIMEOUT_REQUEST"), APITimeoutRequest, config.DurationParser),
			},
			Cors: Cors{
				AllowedOrigins: config.LoadEnv(
					getEnv("API_CORS_ALLOWED_ORIGINS"), APICorsAllowedOrigins, config.SliceParser(config.StringParser),
				),
				AllowedHeaders: config.LoadEnv(
					getEnv("API_CORS_ALLOWED_HEADERS"), APICorsAllowedHeaders, config.SliceParser(config.StringParser),
				),
				AllowCredentials: config.LoadEnv(
					getEnv("API_CORS_ALLOW_CREDENTIALS"), APICorsAllowCredentials, config.BoolParser,
				),
				MaxAge: config.LoadEnv(getEnv("API_CORS_MAX_AGE"), APICorsMaxAge, config.IntParser),
			},
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
		Postgres: PostgresPresetTest,
	}
}
