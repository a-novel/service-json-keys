package cmdpkg

import (
	"os"

	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"

	testutils "github.com/a-novel/service-json-keys/internal/test"
	"github.com/a-novel/service-json-keys/models"
)

func AppConfigTest(port int) AppConfig[*otelpresets.SentryOtelConfig, postgres.Config] {
	return AppConfig[*otelpresets.SentryOtelConfig, postgres.Config]{
		App: AppAppConfig{
			Name:      config.LoadEnv(os.Getenv("APP_NAME"), AppName, config.StringParser),
			MasterKey: os.Getenv("APP_MASTER_KEY"),
		},
		API: AppAPIConfig{
			Port:           port,
			MaxRequestSize: config.LoadEnv(os.Getenv("API_MAX_REQUEST_SIZE"), APIMaxRequestSize, config.Int64Parser),
			Timeouts: AppApiTimeoutsConfig{
				Read: config.LoadEnv(os.Getenv("API_TIMEOUT_READ"), APITimeoutRead, config.DurationParser),
				ReadHeader: config.LoadEnv(
					os.Getenv("API_TIMEOUT_READ_HEADER"), APITimeoutReadHeader, config.DurationParser,
				),
				Write:   config.LoadEnv(os.Getenv("API_TIMEOUT_WRITE"), APITimeoutWrite, config.DurationParser),
				Idle:    config.LoadEnv(os.Getenv("API_TIMEOUT_IDLE"), APITimeoutIdle, config.DurationParser),
				Request: config.LoadEnv(os.Getenv("API_TIMEOUT_REQUEST"), APITimeoutRequest, config.DurationParser),
			},
			Cors: AppCorsConfig{
				AllowedOrigins: config.LoadEnv(
					os.Getenv("API_CORS_ALLOWED_ORIGINS"), APICorsAllowedOrigins, config.SliceParser(config.StringParser),
				),
				AllowedHeaders: config.LoadEnv(
					os.Getenv("API_CORS_ALLOWED_HEADERS"), APICorsAllowedHeaders, config.SliceParser(config.StringParser),
				),
				AllowCredentials: config.LoadEnv(
					os.Getenv("API_CORS_ALLOW_CREDENTIALS"), APICorsAllowCredentials, config.BoolParser,
				),
				MaxAge: config.LoadEnv(os.Getenv("API_CORS_MAX_AGE"), APICorsMaxAge, config.IntParser),
			},
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
		Postgres: testutils.TestDBConfig,
	}
}
