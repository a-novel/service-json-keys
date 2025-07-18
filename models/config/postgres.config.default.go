package config

import (
	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresPresetDefault = postgrespresets.NewDefault(getEnv("POSTGRES_DSN"))
