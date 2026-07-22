// Command rotate-keys rotates active JSON Web Keys. For each configured usage, it generates
// a new key if the rotation interval has elapsed. Consumers see it on their next fetch:
// active_keys is a plain view, so there is no snapshot to refresh.
//
// Designed to run as a periodic job (e.g., a Kubernetes CronJob).
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
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
	daoJwkSearch := dao.NewPgJwkSearch()
	daoJwkInsert := dao.NewPgJwkInsert()

	serviceJwkExtract := core.NewJwkExtract()
	serviceJwkGen := core.NewJwkGen(
		daoJwkSearch,
		daoJwkInsert,
		serviceJwkExtract,
		config.JwkPresetDefault,
	)

	// --- Rotate keys for each usage, as one unit of work ---
	log.Printf("rotating keys for %d configured usage(s)", len(config.JwkPresetDefault))

	serviceJwkRotateAll := core.NewJwkRotateAll(
		serviceJwkGen, postgres.NewTransactor(nil), config.JwkPresetDefault,
	)

	resp, err := serviceJwkRotateAll.Exec(ctx, &core.JwkRotateAllRequest{})
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		log.Fatalln(err.Error()) //nolint:gocritic
	}

	// active_keys is a plain view, so a newly inserted key is visible to the next reader
	// with no refresh step to run — and none to forget.

	otel.ReportSuccessNoContent(span)
	log.Printf("done — %d usage(s) processed, completed in %s",
		resp.Processed, time.Since(start).Round(time.Millisecond))
}
