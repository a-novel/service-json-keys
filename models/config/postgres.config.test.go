package config

import (
	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresPresetTest = postgrespresets.NewDefault(getEnv("POSTGRES_DSN_TEST"))
