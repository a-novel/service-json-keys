package cmdpkg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/a-novel/golib/postgres"
	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/internal/dao"
	testutils "github.com/a-novel/service-json-keys/internal/test"
	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func TestJobRotateKeys(t *testing.T) {
	t.Parallel()

	config := cmdpkg.JobRotateKeysDefault
	config.Postgres = postgrespresets.NewPassthroughConfig(testutils.TestDB)

	postgres.RunIsolatedTransactionalTest(
		t,
		testutils.TestDBConfig,
		func(ctx context.Context, t *testing.T, _ *bun.DB) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			require.NoError(t, cmdpkg.JobRotateKeys(ctx, config))

			// The new keys must also be added to the materialized view.
			// This operation is scheduled regularly in production.
			_, err = db.NewRaw("REFRESH MATERIALIZED VIEW active_keys;").Exec(ctx)
			require.NoError(t, err)

			searchKeysDAO := dao.NewSearchKeysRepository()

			for usage := range config.JWKS {
				keys, err := searchKeysDAO.SearchKeys(ctx, usage)
				require.NoError(t, err)
				assert.Len(t, keys, 1, usage)
			}
		},
	)
}
