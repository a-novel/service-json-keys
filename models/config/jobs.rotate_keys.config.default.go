package config

import (
	"github.com/a-novel/golib/config"
	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

const (
	JobsRotateKeysName = "service-json-keys-job-rotate-keys"
)

var JobRotateKeysPresetDefault = JobRotateKeys[otel.Config, postgres.Config]{
	App: Main{
		Name:      config.LoadEnv(getEnv("APP_NAME"), JobsRotateKeysName, config.StringParser),
		MasterKey: getEnv("APP_MASTER_KEY"),
	},
	JWKS: JWKSPresetDefault,

	Otel:     &OtelDev,
	Postgres: PostgresPresetDefault,
}
