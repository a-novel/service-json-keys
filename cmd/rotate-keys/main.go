// Command rotate-keys generates a new JSON Web Key for each configured usage whose rotation
// interval has elapsed. Consumers pick the new key up on their next fetch.
//
// It runs as a scheduled job, such as a Kubernetes CronJob.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/downtime"
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

	if downtime.Started(cfg.App.DowntimeStart, start) {
		log.Println("planned downtime in progress: no rotation")

		return
	}

	ctx = lo.Must(lib.NewMasterKeyContext(ctx, cfg.App.MasterKey))
	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	ctx, span := otel.Tracer().Start(ctx, "job.RotateKeys")
	defer span.End()

	// --- Wire dependencies ---
	daoJwkSearch := dao.NewPgJwkSearch()
	daoJwkInsert := dao.NewPgJwkInsert()

	serviceJwkExtract := core.NewJwkExtract()
	serviceJwkGen := core.NewJwkGen(daoJwkSearch, daoJwkInsert, serviceJwkExtract, cfg.Jwk)
	serviceJwkRotateAll := core.NewJwkRotateAll(serviceJwkGen, postgres.NewTransactor(nil), cfg.Jwk)

	// --- Rotate keys for each usage, as one unit of work ---
	log.Printf("rotating keys for %d configured usage(s)", len(cfg.Jwk))

	resp, err := serviceJwkRotateAll.Exec(ctx, &core.JwkRotateAllRequest{})
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		log.Fatalln(err.Error()) //nolint:gocritic
	}

	log.Printf("done — %d usage(s) processed, completed in %s",
		resp.Processed, time.Since(start).Round(time.Millisecond))
}
