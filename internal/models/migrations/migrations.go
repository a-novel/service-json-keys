package migrations

import (
	"embed"
)

//go:generate env GOLIB_UPDATE_SNAPSHOTS=1 go test -run ^TestMigrationRoundtrip$

// Migrations embeds all SQL migration files in the package directory.
//
//go:embed *.sql
var Migrations embed.FS
