package handlers_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/a-novel-kit/golib/postgres"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
	"github.com/a-novel/service-json-keys/v2/pkg/go"
)

func TestGrpcStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		skipPostgres  bool
		closePostgres bool
		transaction   bool
		cancel        bool
		expectStatus  codes.Code
	}{
		{name: "Success", expectStatus: codes.OK},
		{name: "Error/MissingPostgres", skipPostgres: true, expectStatus: codes.Unavailable},
		{name: "Error/ClosedPostgres", closePostgres: true, expectStatus: codes.Unavailable},
		{name: "Error/TransactionContext", transaction: true, expectStatus: codes.Unavailable},
		{name: "Error/CancelledProbe", cancel: true, expectStatus: codes.Unavailable},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			if !testCase.skipPostgres {
				var err error

				preset := postgrespresets.NewDefault(configtest.PostgresPreset.Options()...)
				ctx, err = postgres.NewContext(ctx, preset)
				require.NoError(t, err)
				pg, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				db, ok := pg.(*bun.DB)
				require.True(t, ok)
				t.Cleanup(func() { require.NoError(t, db.Close()) })

				if testCase.closePostgres {
					require.NoError(t, db.Close())
				}

				if testCase.transaction {
					tx, err := db.BeginTx(ctx, nil)
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, tx.Rollback()) })

					ctx = context.WithValue(ctx, postgres.ContextKey{}, tx)
				}
			}

			if testCase.cancel {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()

				ctx = cancelled
			}

			response, err := handlers.NewGrpcStatus().Status(ctx, &jsonkeysv2.StatusRequest{})
			result, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, testCase.expectStatus, result.Code())

			if testCase.expectStatus != codes.OK {
				require.Nil(t, response)
				require.Equal(t, "service dependencies unavailable", result.Message())
			} else {
				require.Equal(t, &jsonkeysv2.StatusResponse{
					Postgres: &jsonkeysv2.DependencyHealth{
						Status: jsonkeysv2.DependencyStatus_DEPENDENCY_STATUS_UP,
					},
				}, response)
			}
		})
	}

	t.Run("Transport/Unavailable", func(t *testing.T) {
		t.Parallel()

		listener := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		jsonkeysv2.RegisterStatusServiceServer(server, handlers.NewGrpcStatus())

		serveResult := make(chan error, 1)
		go func() { serveResult <- server.Serve(listener) }()

		t.Cleanup(func() {
			server.Stop()
			require.NoError(t, <-serveResult)
		})

		client, err := servicejsonkeys.NewClient(
			"passthrough:///health-test",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			}),
		)
		require.NoError(t, err)

		defer client.Close()

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		response, err := client.Status(ctx, &servicejsonkeys.StatusRequest{})
		require.Nil(t, response)
		require.Equal(t, codes.Unavailable, status.Code(err))
		require.Equal(t, "service dependencies unavailable", status.Convert(err).Message())
	})
}
