package main

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"
	"github.com/a-novel-kit/golib/servicetoken"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/lib/libtest"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	// The gate refuses a call without a token before it fetches any key, so the issuer never answers.
	gate := servicetoken.Config{
		Issuer:   "https://issuer.test",
		Audience: "service-json-keys",
		Callers:  []string{"system:serviceaccount:agora:caller"},
	}

	testCases := []struct {
		name string

		serviceToken servicetoken.Config
		stream       bool

		expectStatus codes.Code
		expectErr    error
	}{
		{
			// A local stack opts out, and the server takes every call.
			name: "Success/IssuerOptional",

			serviceToken: servicetoken.Config{IssuerOptional: true},

			expectStatus: codes.OK,
		},
		{
			name: "Error/Unauthenticated",

			serviceToken: gate,

			expectStatus: codes.Unauthenticated,
		},
		{
			// Streams pass the same gate.
			name: "Error/UnauthenticatedStream",

			serviceToken: gate,
			stream:       true,

			expectStatus: codes.Unauthenticated,
		},
		{
			// A deployment that forgets the issuer stops at startup rather than admit every caller.
			name: "Error/IssuerRequired",

			expectErr: servicetoken.ErrIssuerRequired,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.AppPresetDefault
			cfg.ServiceToken = testCase.serviceToken

			server, err := newServer(t.Context(), cfg, libtest.MasterKey)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				return
			}

			err = call(t.Context(), serve(t, server), testCase.stream)
			require.Equal(t, testCase.expectStatus, status.Code(err), "got %v", err)
		})
	}
}

// serve serves server over an in-memory listener, and returns a connection to it. The test's
// cleanup stops both.
func serve(t *testing.T, server *grpc.Server) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1 << 16)

	go func() { _ = server.Serve(listener) }()

	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
	)
	if err != nil {
		panic(err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

// call calls golib's echo service on conn, which the gate guards as it guards the service's own: a
// unary call, or a server stream when stream is set. The echo reads no database, so the test needs
// none.
func call(ctx context.Context, conn *grpc.ClientConn, stream bool) error {
	client := golibproto.NewEchoServiceClient(conn)

	if !stream {
		_, err := client.UnaryEcho(ctx, &golibproto.UnaryEchoRequest{})

		return err
	}

	echoStream, err := client.ServerStreamingEcho(ctx, &golibproto.ServerStreamingEchoRequest{})
	if err != nil {
		return err
	}

	_, err = echoStream.Recv()

	return err
}
