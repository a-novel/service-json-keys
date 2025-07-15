package testutils

import (
	"context"
	"os"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel/golib/postgres"
	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/migrations"
)

var TestDBConfig = postgrespresets.DefaultConfig{
	DSN:        os.Getenv("POSTGRES_DSN_TEST"),
	Migrations: migrations.Migrations,
}

var pgCtx = lo.Must(func() (context.Context, error) {
	ctx, err := postgres.InitPostgres(context.Background(), TestDBConfig)

	for i := 0; err != nil && i < 3; i++ {
		ctx, err = postgres.InitPostgres(context.Background(), TestDBConfig) //nolint:fatcontext
	}

	return ctx, err
}())

var TestDB = lo.Must(postgres.GetContext(pgCtx)).(*bun.DB) //nolint:forcetypeassert

const TestMasterKey = "fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"
