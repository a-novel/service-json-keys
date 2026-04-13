package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

// PostgresPresetTest is the PostgreSQL configuration used in integration tests, populated from environment variables.
var PostgresPresetTest = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
