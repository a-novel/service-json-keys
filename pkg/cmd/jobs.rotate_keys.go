package cmdpkg

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
)

type JobRotateKeysConfig[Otel otel.Config, Pg postgres.Config] struct {
	App  AppAppConfig                              `json:"app"  yaml:"app"`
	JWKS map[models.KeyUsage]*models.JSONKeyConfig `json:"jwks" yaml:"jwks"`

	Otel     Otel `json:"otel"     yaml:"otel"`
	Postgres Pg   `json:"postgres" yaml:"postgres"`
}

func JobRotateKeys[Otel otel.Config, Pg postgres.Config](
	ctx context.Context, config JobRotateKeysConfig[Otel, Pg],
) error {
	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================
	otel.SetAppName(config.App.Name)

	err := otel.InitOtel(config.Otel)
	if err != nil {
		return fmt.Errorf("init otel: %w", err)
	}
	defer config.Otel.Flush()

	ctx, err = lib.NewMasterKeyContext(ctx, config.App.MasterKey)
	if err != nil {
		return fmt.Errorf("new master key context: %w", err)
	}

	ctx, err = postgres.InitPostgres(ctx, config.Postgres)
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================
	ctx, span := otel.Tracer().Start(ctx, "job.RotateKeys")
	defer span.End()

	searchKeysDAO := dao.NewSearchKeysRepository()
	insertKeyDAO := dao.NewInsertKeyRepository()

	generateKeysService := services.NewGenerateKeyService(
		services.NewGenerateKeySource(searchKeysDAO, insertKeyDAO),
		config.JWKS,
	)

	var (
		wg    sync.WaitGroup
		sErrs sync.Map
	)

	for usage := range config.JWKS {
		wg.Add(1)

		go func(usage models.KeyUsage) {
			defer wg.Done()

			sErrs.Store(usage, postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				_, localErr := generateKeysService.GenerateKey(ctx, usage)

				return localErr
			}))
		}(usage)
	}

	wg.Wait()

	sErrs.Range(func(key, value interface{}) bool {
		cErr, _ := value.(error)
		err = errors.Join(err, cErr)

		return true
	})

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
	}

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get db from context: %w", err))
	}

	// The new keys must also be added to the materialized view.
	_, err = db.NewRaw("REFRESH MATERIALIZED VIEW active_keys;").Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("refresh materialized view: %w", err))
	}

	span.SetStatus(codes.Ok, "")

	return nil
}
