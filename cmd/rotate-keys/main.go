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
	err := postgres.RunInTx(ctx, nil, func(ctx context.Context, _ bun.IDB) error {
		for usage := range config.JwkPresetDefault {
			_, err := serviceJwkGen.Exec(ctx, &services.JwkGenRequest{Usage: usage})
			if err != nil {
				return fmt.Errorf("generate key for usage %s: %w", usage, err)
			}
		}

		return nil
	})
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		log.Fatalln(err.Error()) //nolint:gocritic
	}

	// --- Refresh the materialized view so consumers see the updated active keys ---
	db := lo.Must(postgres.GetContext(ctx))

	_, err = db.NewRaw("REFRESH MATERIALIZED VIEW active_keys;").Exec(ctx)
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		log.Fatalln(err.Error())
	}

	otel.ReportSuccessNoContent(span)
}
