package migrations

import (
	"embed"
)

//go:embed *.sql

// Migrations embeds all SQL migration files in the package directory.
var Migrations embed.FS
