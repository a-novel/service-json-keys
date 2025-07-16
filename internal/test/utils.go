package testutils

import (
	"os"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var TestDBConfig = postgrespresets.NewDefault(os.Getenv("POSTGRES_DSN_TEST"))

const TestMasterKey = "fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"
