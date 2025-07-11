package cmdpkg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/internal/dao"
	testutils "github.com/a-novel/service-json-keys/internal/test"
	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func TestJobRotateKeys(t *testing.T) {
	t.Parallel()

	config := cmdpkg.JobRotateKeysDefault
	config.Postgres = postgrespresets.NewPassthroughConfig(testutils.TestDB)

	testutils.TransactionalTest(t, "Job.GenerateKeys", func(ctx context.Context, t *testing.T) {
		t.Helper()

		require.NoError(t, cmdpkg.JobRotateKeys(ctx, config))

		searchKeysDAO := dao.NewSearchKeysRepository()

		for usage := range config.JWKS {
			keys, err := searchKeysDAO.SearchKeys(ctx, usage)
			require.NoError(t, err)
			require.Len(t, keys, 1)
		}
	})
}
