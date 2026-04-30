// Package configtest holds shared test fixtures for the config package. Production code must
// not import it; only test files (in this or any other package's `_test.go` files) may.
//
// The fixtures live in their own subpackage so they cannot be compiled into the production
// binary. A file named `*.test.go` in the production `config` package would still ship in the
// binary because Go's exclusion rule only matches `_test.go` (with an underscore).
package configtest

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

// PostgresPreset is the PostgreSQL configuration used in integration tests, populated from
// environment variables.
var PostgresPreset = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
