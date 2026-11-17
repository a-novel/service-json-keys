package servicejsonkeys_test

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/a-novel-kit/golib/grpcf"
	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

func TestClient(t *testing.T) {
	t.Parallel()

	t.Run("Success/DependencyBoundary", func(t *testing.T) {
		t.Parallel()

		const module = "github.com/a-novel/service-json-keys/v2/"

		command := exec.CommandContext(t.Context(), "go", "list", "-deps", ".")
		output, err := command.Output()
		require.NoError(t, err)

		var internalDependencies []string

		for dependency := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
			if strings.HasPrefix(dependency, module) && strings.Contains(dependency, "/internal/") {
				internalDependencies = append(internalDependencies, strings.TrimPrefix(dependency, module))
			}
		}

		// The client links neither the server's environment nor any service layer, so importing it
		// reads no server configuration.
		require.ElementsMatch(t, []string{
			"internal/config/jwkconfig",
			"internal/handlers/protogen/anovel/jsonkeys/v2",
			"pkg/go/internal/tokens",
		}, internalDependencies)
	})

	t.Run("Success/PropagatesTraceContext", func(t *testing.T) {
		t.Parallel()

		// golib's otel.Init installs this propagator in every service; the default one sends nothing.
		otel.SetTextMapPropagator(propagation.TraceContext{})

		received := make(chan []string, 1)
		server := grpc.NewServer(grpc.UnaryInterceptor(func(
			ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
		) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			received <- md.Get("traceparent")

			return handler(ctx, req)
		}))
		grpcf.RegisterEchoServers(server)

		listener := bufconn.Listen(1 << 16)
		go func() { _ = server.Serve(listener) }()

		defer server.Stop()

		client, err := servicejsonkeys.NewClient(
			"passthrough:///bufconn",
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)

		defer client.Close()

		callerConfig := trace.SpanContextConfig{
			TraceID:    trace.TraceID{1},
			SpanID:     trace.SpanID{1},
			TraceFlags: trace.FlagsSampled,
		}
		caller := trace.NewSpanContext(callerConfig)

		_, err = client.UnaryEcho(
			trace.ContextWithSpanContext(t.Context(), caller), &golibproto.UnaryEchoRequest{},
		)
		require.NoError(t, err)

		traceparent := <-received
		require.Len(t, traceparent, 1)
		require.Contains(t, traceparent[0], caller.TraceID().String())
	})

	t.Run("Success/RemoteCalls", func(t *testing.T) {
		t.Parallel()

		client, err := servicejsonkeys.NewClient(
			configtest.GrpcURL,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)

		defer client.Close()

		_, err = client.UnaryEcho(t.Context(), &golibproto.UnaryEchoRequest{})
		require.NoError(t, err)

		health, err := client.Status(t.Context(), &servicejsonkeys.StatusRequest{})
		require.NoError(t, err)
		require.NotNil(t, health.GetPostgres())

		keys, err := client.JwkList(t.Context(), &servicejsonkeys.JwkListRequest{
			Usage: servicejsonkeys.KeyUsageAuth,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(keys.GetKeys()), 1)

		key, err := client.JwkGet(t.Context(), &servicejsonkeys.JwkGetRequest{
			Id: keys.GetKeys()[0].GetKid(),
		})
		require.NoError(t, err)
		require.Equal(t, key.GetJwk(), keys.GetKeys()[0])
	})
}
