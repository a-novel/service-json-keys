// Command migrations applies pending SQL migrations to the JSON Keys database. Run it on first
// deploy and after each schema change.
package main

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/models/migrations"
)

func main() {
	ctx := context.Background()

	db := lo.Must(config.PostgresPresetMigrations.DB(ctx))
	defer config.PostgresPresetMigrations.Close()

	lo.Must0(postgres.RunMigrations(ctx, db, migrations.Migrations))
}
