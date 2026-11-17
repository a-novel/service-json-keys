package config

import (
	"github.com/a-novel/service-json-keys/v2/internal/config/env"
	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
)

// JobRotateKeysPresetDefault is the default [JobRotateKeys] configuration populated from environment variables.
var JobRotateKeysPresetDefault = JobRotateKeys{
	App: Main{
		Name:          env.AppName + "-job-rotate-keys",
		MasterKey:     env.AppMasterKey,
		DowntimeStart: env.DowntimeStart,
	},
	Jwk: jwkconfig.PresetDefault,

	Otel:     OtelJobPresetDefault,
	Postgres: PostgresPresetDefault,
}
