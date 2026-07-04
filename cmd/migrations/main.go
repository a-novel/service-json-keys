// Command migrations applies pending SQL migrations to the JSON-keys database.
// Run this once on first deploy and after each schema change.
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

	// Inventory the .up.sql files up front for the start-of-run log. The
	// bun migrator doesn't expose a count, so the embedded FS is the closest
	// stable source.
	discovered := listUpMigrations(migrations.Migrations)
	log.Printf("discovered %d migration(s) in models/migrations", len(discovered))

	for _, name := range discovered {
		log.Printf("  · %s", name)
	}

	log.Println("connecting to database...")

	ctx := lo.Must(postgres.NewContext(context.Background(), config.PostgresPresetDefault))

	log.Println("applying pending migrations...")
	lo.Must0(postgres.RunMigrationsContext(ctx, migrations.Migrations))

	log.Printf("done — %d migration(s) examined, completed in %s",
		len(discovered), time.Since(start).Round(time.Millisecond))
}

// listUpMigrations returns the bare name of every *.up.sql in the migrations
// FS, in lexical order (the timestamp prefix makes that chronological). It
// feeds the start-of-run inventory log only; bun's migrator decides which
// migrations actually run.
func listUpMigrations(f fs.FS) []string {
	var out []string

	_ = fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Aborts the walk on error; the caller discards it, since the
			// inventory is best-effort logging rather than a gate.
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".up.sql") {
			out = append(out, strings.TrimSuffix(path, ".up.sql"))
		}

		return nil
	})

	return out
}
