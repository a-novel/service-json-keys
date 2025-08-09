package config

import (
	"github.com/a-novel/golib/config"
	otelpresets "github.com/a-novel/golib/otel/presets"
	"github.com/a-novel/golib/postgres"
)

var JobRotateKeysPresetTest = JobRotateKeys[*otelpresets.LocalOtelConfig, postgres.Config]{
	App: Main{
		Name:      config.LoadEnv(getEnv("APP_NAME"), JobsRotateKeysName, config.StringParser),
		MasterKey: getEnv("APP_MASTER_KEY"),
	},
	JWKS: JWKSPresetDefault,

	Otel:     &OtelDev,
	Postgres: PostgresPresetTest,
}
