package configtest

import (
	"github.com/a-novel/service-json-keys/v2/internal/config"
)

// PostgresPreset is the PostgreSQL configuration of integration tests. It is the production
// preset, so tests track it with no second definition.
var PostgresPreset = config.PostgresPresetDefault
