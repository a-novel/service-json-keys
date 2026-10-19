package config

import "github.com/a-novel/service-json-keys/v2/internal/config/env"

// JobRotateKeysPresetDefault is the default [JobRotateKeys] configuration populated from environment variables.
var JobRotateKeysPresetDefault = JobRotateKeys{
	App: Main{
		Name:          env.AppName + "-job-rotate-keys",
		MasterKey:     env.AppMasterKey,
		DowntimeStart: env.DowntimeStart,
	},
	Jwk: JwkPresetDefault,

	Otel:     OtelPresetDefault,
	Postgres: PostgresPresetDefault,
}
