// Command migrations applies pending SQL migrations to the JSON Keys database. Run it on first
// deploy and after each schema change.
package main

import (
	"context"
	"io/fs"
	"log"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/models/migrations"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("migrations: ")

	start := time.Now()

	// The bun migrator exposes no count, so the start-of-run log lists the embedded files. The
	// timestamp prefix sorts them in order.
	discovered := lo.Must(fs.Glob(migrations.Migrations, "*.up.sql"))
	log.Printf("discovered %d migration(s) in models/migrations", len(discovered))

	for _, name := range discovered {
		log.Printf("  · %s", strings.TrimSuffix(name, ".up.sql"))
	}

	log.Println("connecting to database...")

	ctx := lo.Must(postgres.NewContext(context.Background(), config.PostgresPresetMigrations))

	log.Println("applying pending migrations...")
	lo.Must0(postgres.RunMigrationsContext(ctx, migrations.Migrations))

	log.Printf("done — %d migration(s) examined, completed in %s",
		len(discovered), time.Since(start).Round(time.Millisecond))
}
