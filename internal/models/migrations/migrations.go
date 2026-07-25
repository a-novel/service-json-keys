package migrations

import (
	"embed"
)

//go:generate env GOLIB_UPDATE_SNAPSHOTS=1 go test -run ^TestMigrationRoundtrip$
//go:embed *.sql

// Migrations embeds all SQL migration files in the package directory.
var Migrations embed.FS
