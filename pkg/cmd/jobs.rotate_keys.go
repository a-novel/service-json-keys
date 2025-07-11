package cmdpkg

import (
	"context"
	"fmt"

	"github.com/go-faster/errors"
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

	rotateKeyUsage := func(usage models.KeyUsage) error {
		_, err = generateKeysService.GenerateKey(ctx, usage)
		if err != nil {
			return err
		}

		return nil
	}

	for usage := range config.JWKS {
		err = errors.Join(rotateKeyUsage(usage), err)
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
	}

	span.SetStatus(codes.Ok, "")

	return nil
}
