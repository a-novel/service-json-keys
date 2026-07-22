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
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

// jwkGenerator ensures a key exists for one usage, rotating it when its interval
// has elapsed. Declared here rather than taken as a concrete type so the rotation
// can be exercised without a key service behind it.
type jwkGenerator interface {
	Exec(ctx context.Context, request *core.JwkGenRequest) (*core.Jwk, error)
}

// rotateKeys ensures every configured usage has a current key, as a single unit of
// work: a failure partway through leaves none of them rotated.
//
// That is the whole reason it takes a transactor rather than reaching for one. The
// rotation used to open a transaction and then hand each generation the surrounding
// context, so every key committed on its own and a failure on the third usage left
// the first two rotated — with the job reporting failure as though nothing had been
// written.
//
// It returns the number of usages processed, which is only meaningful when the error
// is nil: a partial count belongs to work that has been rolled back.
func rotateKeys(
	ctx context.Context,
	transactor transaction.Transactor,
	generator jwkGenerator,
	usages map[string]*config.Jwk,
) (int, error) {
	processed := 0

	err := transactor.WithinTx(ctx, func(ctx context.Context) error {
		for usage := range usages {
			log.Printf("  · %s: ensuring key (rotated if interval elapsed)", usage)

			_, err := generator.Exec(ctx, &core.JwkGenRequest{Usage: usage})
			if err != nil {
				return fmt.Errorf("generate key for usage %s: %w", usage, err)
			}

			processed++
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return processed, nil
}

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

	processed, err := rotateKeys(ctx, postgres.NewTransactor(nil), serviceJwkGen, config.JwkPresetDefault)
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		log.Fatalln(err.Error()) //nolint:gocritic
	}

	// active_keys is a plain view, so a newly inserted key is visible to the next reader
	// with no refresh step to run — and none to forget.

	otel.ReportSuccessNoContent(span)
	log.Printf("done — %d usage(s) processed, completed in %s",
		processed, time.Since(start).Round(time.Millisecond))
}
