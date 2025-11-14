package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/internal/config"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	"github.com/a-novel/service-json-keys/internal/services"
)

func main() {
	cfg := config.JobRotateKeysPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	ctx = lo.Must(lib.NewMasterKeyContext(ctx, cfg.App.MasterKey))
	ctx = lo.Must(postgres.NewContext(ctx, config.PostgresPresetDefault))

	ctx, span := otel.Tracer().Start(ctx, "job.RotateKeys")
	defer span.End()

	repositoryJwkSearch := dao.NewJwkSearch()
	repositoryJwkInsert := dao.NewJwkInsert()

	serviceJwkExtract := services.NewJwkExtract()
	serviceJwkGen := services.NewJwkGen(
		repositoryJwkSearch,
		repositoryJwkInsert,
		serviceJwkExtract,
		config.JwkPresetDefault,
	)

	var err error

	for usage := range config.JwkPresetDefault {
		err = errors.Join(err, postgres.RunInTx(ctx, nil, func(ctx context.Context, _ bun.IDB) error {
			_, localErr := serviceJwkGen.Exec(ctx, &services.JwkGenRequest{Usage: usage})

			return localErr
		}))
	}

	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		span.End()
		log.Fatalln(err.Error()) //nolint:gocritic

		return
	}

	db := lo.Must(postgres.GetContext(ctx))

	// The new keys must also be added to the materialized view.
	_, err = db.NewRaw("REFRESH MATERIALIZED VIEW active_keys;").Exec(ctx)
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
		span.End()
		log.Fatalln(err.Error())

		return
	}

	otel.ReportSuccessNoContent(span)
}
