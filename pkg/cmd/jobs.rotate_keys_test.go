package cmdpkg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/internal/dao"
	testutils "github.com/a-novel/service-json-keys/internal/test"
	"github.com/a-novel/service-json-keys/migrations"
	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func TestJobRotateKeys(t *testing.T) {
	t.Parallel()

	postgres.RunIsolatedTransactionalTest(
		t,
		testutils.TestDBConfig,
		migrations.Migrations,
		func(ctx context.Context, t *testing.T) {
			t.Helper()

			require.NoError(t, cmdpkg.JobRotateKeys(ctx, cmdpkg.JobRotateKeysConfigTest))

			searchKeysDAO := dao.NewSearchKeysRepository()

			for usage := range cmdpkg.JobRotateKeysConfigTest.JWKS {
				keys, err := searchKeysDAO.SearchKeys(ctx, usage)
				require.NoError(t, err)
				assert.Len(t, keys, 1, usage)
			}
		},
	)
}
