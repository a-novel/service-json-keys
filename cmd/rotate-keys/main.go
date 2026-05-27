// Command rotate-keys rotates active JSON Web Keys. For each configured usage, it generates
// a new key if the rotation interval has elapsed, then refreshes the active_keys materialized
// view so consumers see the updated set immediately.
//
// Designed to run as a periodic job (e.g., a Kubernetes CronJob).
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("rotate-keys: ")
	start := time.Now()

	// --- Bootstrap: load config, init telemetry and context ---
	cfg := config.JobRotateKeysPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	ctx = lo.Must(lib.NewMasterKeyContext(ctx, cfg.App.MasterKey))
	ctx = lo.Must(postgres.NewContext(ctx, config.PostgresPresetDefault))

	ctx, span := otel.Tracer().Start(ctx, "job.RotateKeys")
	defer span.End()

	// --- Wire dependencies ---
	repositoryJwkSearch := dao.NewPgJwkSearch()
	repositoryJwkInsert := dao.NewPgJwkInsert()

	serviceJwkExtract := services.NewJwkExtract()
	serviceJwkGen := services.NewJwkGen(
		repositoryJwkSearch,
		repositoryJwkInsert,
		serviceJwkExtract,
		config.JwkPresetDefault,
	)

	// --- Rotate keys for each usage (inside a transaction for atomicity) ---
	log.Printf("rotating keys for %d configured usage(s)", len(config.JwkPresetDefault))
	processed := 0
	err := postgres.RunInTx(ctx, nil, func(ctx context.Context, _ bun.IDB) error {
		for usage := range config.JwkPresetDefault {
			log.Printf("  · %s: ensuring key (rotated if interval elapsed)", usage)
			_, err := serviceJwkGen.Exec(ctx, &services.JwkGenRequest{Usage: usage})
			if err != nil {
				return fmt.Errorf("generate key for usage %s: %w", usage, err)
			}
			processed++
		}

		return nil
	})
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		log.Fatalln(err.Error()) //nolint:gocritic
	}

	// --- Refresh the materialized view so consumers see the updated active keys ---
	log.Println("refreshing active_keys materialized view...")
	db := lo.Must(postgres.GetContext(ctx))

	// CONCURRENTLY allows reads to continue during the refresh; requires the unique index
	// on active_keys.id added by the 20260416000000_active_keys_unique_index migration.
	// Must run outside any transaction block — postgres.RunInTx has already committed above.
	_, err = db.NewRaw("REFRESH MATERIALIZED VIEW CONCURRENTLY active_keys;").Exec(ctx)
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		log.Fatalln(err.Error())
	}

	otel.ReportSuccessNoContent(span)
	log.Printf("done — %d usage(s) processed, completed in %s",
		processed, time.Since(start).Round(time.Millisecond))
}
