package main

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"
	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
	"github.com/a-novel/service-json-keys/v2/internal/lib/libtest"
)

// apiKeyPEM is the public key of RFC 8037 appendix A.2, as a setting of system API keys holds it.
const apiKeyPEM = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEA11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=
-----END PUBLIC KEY-----
`

func TestNewServer(t *testing.T) {
	t.Parallel()

	const (
		audience = "service-json-keys"
		caller   = "system:serviceaccount:agora:caller"
	)

	issuer := servicetokentest.NewIssuer(t)
	gate := servicetoken.Config{Issuer: issuer.URL, Audience: audience, Callers: []string{caller}}

	testCases := []struct {
		name string

		serviceToken servicetoken.Config
		apiKeys      map[string]string
		call         func(ctx context.Context, conn *grpc.ClientConn) error
		token        string

		expectStatus codes.Code
		expectErr    error
	}{
		{
			// A caller's token passes the gate.
			name: "Success",

			serviceToken: gate,
			call:         unaryEcho,
			token:        issuer.Sign(t, issuer.Claims(caller, audience)),

			expectStatus: codes.OK,
		},
		{
			// The kubelet's probe reaches the health service without a token.
			name: "Success/Health",

			serviceToken: gate,
			call:         checkHealth,

			expectStatus: codes.OK,
		},
		{
			// A local stack opts out, and the server takes every call.
			name: "Success/IssuerOptional",

			serviceToken: servicetoken.Config{IssuerOptional: true},
			call:         unaryEcho,

			expectStatus: codes.OK,
		},
		{
			// The echo service carries no auth of its own, so the refusal can only come from the gate.
			name: "Error/Unauthenticated",

			serviceToken: gate,
			call:         unaryEcho,

			expectStatus: codes.Unauthenticated,
		},
		{
			// Streams pass the same gate.
			name: "Error/UnauthenticatedStream",

			serviceToken: gate,
			call:         streamEcho,

			expectStatus: codes.Unauthenticated,
		},
		{
			// A deployment that forgets the issuer stops at startup rather than admit every caller.
			name: "Error/IssuerRequired",

			expectErr: servicetoken.ErrIssuerRequired,
		},
		{
			// The server serves the system API keys of its settings.
			name: "Success/APIKey",

			serviceToken: servicetoken.Config{IssuerOptional: true},
			apiKeys:      map[string]string{"test-purpose": apiKeyPEM},
			call:         apiKeyGet,

			expectStatus: codes.OK,
		},
		{
			// A setting that holds anything but public keys stops the server at startup.
			name: "Error/InvalidAPIKeys",

			serviceToken: servicetoken.Config{IssuerOptional: true},
			apiKeys:      map[string]string{"test-purpose": "not a key"},

			expectErr: core.ErrInvalidAPIKeys,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.AppPresetDefault
			cfg.ServiceToken = testCase.serviceToken
			cfg.APIKeys = testCase.apiKeys

			server, err := newServer(t.Context(), cfg, libtest.MasterKey)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				return
			}

			ctx := t.Context()
			if testCase.token != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, servicetoken.Header, testCase.token)
			}

			err = testCase.call(ctx, serve(t, server))
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

// apiKeyGet reads the key of [apiKeyPEM] by its ID, the thumbprint RFC 8037 appendix A.3 gives.
func apiKeyGet(ctx context.Context, conn *grpc.ClientConn) error {
	_, err := jsonkeysv2.NewAPIKeyGetServiceClient(conn).APIKeyGet(ctx, &jsonkeysv2.APIKeyGetRequest{
		Id: "kPrK_qmxVWaYVA9wwBF6Iuo3vVzz7TxHCTwXBygrS4k",
	})

	return err
}

// unaryEcho calls golib's echo service through a unary call. The gate guards it as it guards the
// service's own, and it reads no database, so the test needs none.
func unaryEcho(ctx context.Context, conn *grpc.ClientConn) error {
	_, err := golibproto.NewEchoServiceClient(conn).UnaryEcho(ctx, &golibproto.UnaryEchoRequest{})

	return err
}

// streamEcho calls golib's echo service through a server stream, and reads its first message.
func streamEcho(ctx context.Context, conn *grpc.ClientConn) error {
	stream, err := golibproto.NewEchoServiceClient(conn).ServerStreamingEcho(
		ctx, &golibproto.ServerStreamingEchoRequest{},
	)
	if err != nil {
		return err
	}

	_, err = stream.Recv()

	return err
}

// checkHealth asks the health service for the server's status, as the kubelet's probe does. A status
// other than SERVING fails without a gRPC code.
func checkHealth(ctx context.Context, conn *grpc.ClientConn) error {
	res, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return err
	}

	if res.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("health status %s", res.GetStatus())
	}

	return nil
}
