package cmdpkg_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	testutils "github.com/a-novel/service-json-keys/internal/test"
	"github.com/a-novel/service-json-keys/migrations"
	"github.com/a-novel/service-json-keys/models/api"
	"github.com/a-novel/service-json-keys/pkg"
	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func TestApp(t *testing.T) {
	t.Parallel()

	testSuites := map[string]func(ctx context.Context, t *testing.T, client *apimodels.Client){
		"Ping":          testAppPing,
		"SignAndVerify": testAppSignAndVerify,
	}

	for testName, testSuite := range testSuites {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			listener, err := net.Listen("tcp", ":0")
			require.NoError(t, err)

			addr, ok := listener.Addr().(*net.TCPAddr)
			require.True(t, ok, "expected TCPAddr, got %T", listener.Addr())

			port := addr.Port

			// Close the listener.
			require.NoError(t, listener.Close(), "failed to close listener")

			postgres.RunIsolatedTransactionalTest(
				t, testutils.TestDBConfig, migrations.Migrations, func(ctx context.Context, t *testing.T) {
					t.Helper()

					require.NoError(t, cmdpkg.JobRotateKeys(ctx, cmdpkg.JobRotateKeysConfigTest))

					appConfig := cmdpkg.AppConfigTest(port)

					go func() {
						assert.NoError(t, cmdpkg.App(ctx, appConfig))
					}()

					client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", port))
					require.NoError(t, err)

					testSuite(ctx, t, client)
				},
			)
		})
	}
}
