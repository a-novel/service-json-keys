// Package configtest holds shared test fixtures for the config package. Only `_test.go` files
// may import it.
//
// Isolation is a convention the language does not enforce: Go links any package reachable from
// a production import into the binary, so keeping configtest out of every production import
// path is what confines these fixtures to the test binary. A dedicated subpackage makes that
// boundary easy to lint or grep against.
package configtest

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

// PostgresPreset is the PostgreSQL configuration used in integration tests, populated from
// environment variables.
var PostgresPreset = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
