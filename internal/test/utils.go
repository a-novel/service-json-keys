package testutils

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/a-novel/golib/postgres"
	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/migrations"
)

var pgCtx = lo.Must(func() (context.Context, error) {
	ctx, err := postgres.InitPostgres(context.Background(), postgrespresets.DefaultConfig{
		DSN:        os.Getenv("POSTGRES_DSN_TEST"),
		Migrations: migrations.Migrations,
	})

	// This function can be run multiple times across modules, so migration conflicts can happen.
	for i := 0; i < 3 && err != nil; i++ {
		ctx, err = postgres.InitPostgres(context.Background(), postgrespresets.DefaultConfig{ //nolint:fatcontext
			DSN:        os.Getenv("POSTGRES_DSN_TEST"),
			Migrations: migrations.Migrations,
		})
	}

	return ctx, err
}())

var TestDB = lo.Must(postgres.GetContext(pgCtx)).(*bun.DB) //nolint:forcetypeassert

const TestMasterKey = "fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"

func TransactionalTest(t *testing.T, name string, callback func(ctx context.Context, t *testing.T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Parallel()

		tx, err := TestDB.BeginTx(t.Context(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		require.NoError(t, err)

		ctx := context.WithValue(t.Context(), postgres.ContextKey{}, tx)
		t.Cleanup(func() {
			_ = tx.Rollback()
		})

		callback(ctx, t)
	})
}
