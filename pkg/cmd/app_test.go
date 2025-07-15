package cmdpkg_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/a-novel/golib/config"
	"github.com/a-novel/golib/postgres"
	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
	testutils "github.com/a-novel/service-json-keys/internal/test"
	"github.com/a-novel/service-json-keys/pkg"
	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func TestApp(t *testing.T) {
	t.Parallel()

	apiConfig := cmdpkg.AppConfigDefault
	apiConfig.API.Port = config.LoadEnv(os.Getenv("API_PORT_TEST"), 0, config.IntParser)
	apiConfig.Postgres = postgrespresets.NewPassthroughConfig(testutils.TestDB)

	rotateKeysConfig := cmdpkg.JobRotateKeysDefault
	rotateKeysConfig.Postgres = postgrespresets.NewPassthroughConfig(testutils.TestDB)

	postgres.RunIsolatedTransactionalTest(
		t,
		testutils.TestDBConfig,
		func(ctx context.Context, t *testing.T, _ *bun.DB) {
			t.Helper()

			require.NoError(t, cmdpkg.JobRotateKeys(ctx, rotateKeysConfig))

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			// The new keys must also be added to the materialized view.
			// This operation is scheduled regularly in production.
			_, err = db.NewRaw("REFRESH MATERIALIZED VIEW active_keys;").Exec(ctx)
			require.NoError(t, err)

			go func() {
				assert.NoError(t, cmdpkg.App(ctx, apiConfig))
			}()

			client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", apiConfig.API.Port))
			require.NoError(t, err)

			testSuites := map[string]func(ctx context.Context, t *testing.T, client *codegen.Client){
				"Ping":          testAppPing,
				"SignAndVerify": testAppSignAndVerify,
			}

			for testName, testSuite := range testSuites {
				t.Run(testName, func(t *testing.T) {
					t.Parallel()
					testSuite(ctx, t, client)
				})
			}
		},
	)
}
