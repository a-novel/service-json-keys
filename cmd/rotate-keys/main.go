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
	"github.com/a-novel-kit/golib/sentryf"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("rotate-keys: ")

	err := rotateKeys()
	if err != nil {
		log.Fatalln(err.Error())
	}
}

// rotateKeys runs the job. It returns its error to main, so the job's span ends and its telemetry
// and error report flush before the process exits.
func rotateKeys() error {
	start := time.Now()

	// --- Bootstrap: load config, init telemetry ---
	cfg := config.JobRotateKeysPresetDefault

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	lo.Must0(sentryf.Init(cfg.Sentry))

	defer sentryf.Flush(config.SentryJobFlushTimeout)

	if downtime.Started(cfg.App.DowntimeStart, start) {
		log.Println("planned downtime in progress: no rotation")

		return nil
	}

	return run(context.Background(), cfg, start)
}

// run rotates the keys under the job's span, and reports its failure to Sentry, a failure to load
// the master key or to reach the database included.
func run(ctx context.Context, cfg config.JobRotateKeys, start time.Time) error {
	ctx, span := otel.Tracer().Start(ctx, "job.RotateKeys")
	defer span.End()

	err := rotate(ctx, cfg, start)
	if err != nil {
		err = otel.ReportError(span, err)
		sentryf.Capture(ctx, "rotate-keys", err)
	}

	return err
}

// rotate connects to the database, then rotates the keys of each configured usage as one unit of
// work.
func rotate(ctx context.Context, cfg config.JobRotateKeys, start time.Time) error {
	ctx, err := lib.NewMasterKeyContext(ctx, cfg.App.MasterKey)
	if err != nil {
		return fmt.Errorf("load master key: %w", err)
	}

	ctx, err = postgres.NewContext(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

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
		return fmt.Errorf("rotate keys: %w", err)
	}

	log.Printf("done — %d usage(s) processed, completed in %s",
		resp.Processed, time.Since(start).Round(time.Millisecond))

	return nil
}
