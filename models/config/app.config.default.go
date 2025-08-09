package config

import (
	"time"

	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	"github.com/a-novel/golib/otel"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
)

const (
	OtelFlushTimeout = 2 * time.Second

	AppName = "service-json-keys"

	APIPort                 = 8080
	APITimeoutRead          = 5 * time.Second
	APITimeoutReadHeader    = 3 * time.Second
	APITimeoutWrite         = 10 * time.Second
	APITimeoutIdle          = 30 * time.Second
	APITimeoutRequest       = 15 * time.Second
	APIMaxRequestSize       = 2 << 20 // 2 MiB
	APICorsAllowCredentials = false
	APICorsMaxAge           = 3600
)

var (
	APICorsAllowedOrigins = []string{"*"}
	APICorsAllowedHeaders = []string{"*"}
)

var OtelProd = otelpresets.GCloudOtelConfig{
	ProjectID:    getEnv("GCLOUD_PROJECT_ID"),
	FlushTimeout: OtelFlushTimeout,
}

var OtelDev = otelpresets.LocalOtelConfig{
	PrettyPrint:  config.LoadEnv(getEnv("PRETTY_CONSOLE"), true, config.BoolParser),
	FlushTimeout: OtelFlushTimeout,
}

var AppPresetDefault = App[otel.Config, postgres.Config]{
	App: Main{
		Name:      config.LoadEnv(getEnv("APP_NAME"), AppName, config.StringParser),
		MasterKey: getEnv("APP_MASTER_KEY"),
	},
	API: API{
		Port:           config.LoadEnv(getEnv("API_PORT"), APIPort, config.IntParser),
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

	Otel:     lo.Ternary[otel.Config](getEnv("GCLOUD_PROJECT_ID") == "", &OtelDev, &OtelProd),
	Postgres: PostgresPresetDefault,
}
