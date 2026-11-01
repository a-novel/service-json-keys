package handlers_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
	"github.com/a-novel/service-json-keys/v2/pkg/go"
)

func TestGrpcStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		setup probeSetup

		expect       *jsonkeysv2.StatusResponse
		expectStatus codes.Code
	}{
		{
			name:  "Success",
			setup: probeHealthy,
			expect: &jsonkeysv2.StatusResponse{
				Postgres: &jsonkeysv2.DependencyHealth{
					Status: jsonkeysv2.DependencyStatus_DEPENDENCY_STATUS_UP,
				},
			},
			expectStatus: codes.OK,
		},
		{name: "Error/MissingPostgres", setup: probeWithoutPostgres, expectStatus: codes.Unavailable},
		{name: "Error/ClosedPostgres", setup: probeClosedPostgres, expectStatus: codes.Unavailable},
		{name: "Error/TransactionContext", setup: probeInTransaction, expectStatus: codes.Unavailable},
		{name: "Error/CancelledProbe", setup: probeCancelled, expectStatus: codes.Unavailable},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := newProbeContext(t, testCase.setup)

			response, err := handlers.NewGrpcStatus().Status(ctx, &jsonkeysv2.StatusRequest{})
			result, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, testCase.expectStatus, result.Code(), "got %s (%v)", result.Code(), err)
			require.Equal(t, testCase.expect, response)

			if testCase.expectStatus != codes.OK {
				// The message carries no raw dependency error. The fault keeps that error, so the server
				// span names its type.
				require.Equal(t, "service dependencies unavailable", result.Message())
				require.Error(t, errors.Unwrap(err))
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
