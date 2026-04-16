package servicejsonkeys

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

type (
	StatusRequest      = protogen.StatusRequest
	StatusResponse     = protogen.StatusResponse
	JwkListRequest     = protogen.JwkListRequest
	JwkListResponse    = protogen.JwkListResponse
	JwkGetRequest      = protogen.JwkGetRequest
	JwkGetResponse     = protogen.JwkGetResponse
	ClaimsSignRequest  = protogen.ClaimsSignRequest
	ClaimsSignResponse = protogen.ClaimsSignResponse

	// JwkPublicSources holds the per-usage public-key sources for all supported algorithm families.
	// Retrieved via [Client.Sources].
	JwkPublicSources = services.JwkPublicSources
	// JwkRecipients holds the per-usage JWT verification plugins.
	// Retrieved via [Client.Recipients].
	JwkRecipients = services.JwkRecipients
	// JwkConfig holds the full configuration for a single key usage: signing algorithm,
	// key lifetime and caching parameters, and JWT token claim parameters.
	// Keyed by usage name in the map returned by [Client.Keys].
	JwkConfig = config.Jwk
)

// BaseClient is the minimal gRPC interface for the JSON-keys service. It exposes every RPC
// endpoint and a Close method to release the underlying connection.
//
// Prefer [Client] when you need the pre-built JWT verification helpers. Use BaseClient
// when you only need raw RPC access or want to write a lightweight adapter or mock.
type BaseClient interface {
	// UnaryEcho sends a ping to the server and expects the same payload back.
	// Use it to health-check the connection before sending application requests.
	UnaryEcho(
		ctx context.Context, req *golibproto.UnaryEchoRequest, opts ...grpc.CallOption,
	) (*golibproto.UnaryEchoResponse, error)
	// Status reports the operational health of the service and its dependencies.
	Status(ctx context.Context, req *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)

	// JwkGet retrieves a single JSON Web Key by its key ID.
	JwkGet(ctx context.Context, req *JwkGetRequest, opts ...grpc.CallOption) (*JwkGetResponse, error)
	// JwkList returns the active public keys for a given usage.
	JwkList(ctx context.Context, req *JwkListRequest, opts ...grpc.CallOption) (*JwkListResponse, error)
	// ClaimsSign asks the service to sign claims and returns a compact JWT.
	// The request payload must be a protobuf Any wrapping the claims to embed;
	// the response token is the resulting compact JWT string.
	ClaimsSign(ctx context.Context, req *ClaimsSignRequest, opts ...grpc.CallOption) (*ClaimsSignResponse, error)

	// Close releases the underlying gRPC connection. Typically called via defer after NewClient.
	Close()
}

// Client extends [BaseClient] with pre-built JWT verification infrastructure. On creation,
// it initializes typed, cached key sources for each configured usage, enabling local token
// verification without a network call per token. Keys are fetched lazily on first use
// and refreshed according to each usage's configured cache duration.
//
// Obtain a Client with [NewClient]. Pass it to [NewClaimsVerifier] to verify tokens.
type Client interface {
	BaseClient

	// Sources returns the per-usage public key sources initialized at construction time.
	// Keys are fetched lazily on first use and cached for each usage's configured cache duration.
	Sources() *JwkPublicSources
	// Recipients returns the per-usage JWT recipient plugins for local token verification,
	// keyed by usage name (e.g., [KeyUsageAuth]).
	Recipients() JwkRecipients
	// Keys returns the JWK configuration map this client was built with, keyed by usage name.
	Keys() map[string]*JwkConfig
}

type client struct {
	golibproto.EchoServiceClient
	protogen.StatusServiceClient
	protogen.JwkGetServiceClient
	protogen.JwkListServiceClient
	protogen.ClaimsSignServiceClient

	sources    *JwkPublicSources
	recipients JwkRecipients
	keys       map[string]*JwkConfig

	conn *grpc.ClientConn
}

func (c *client) Sources() *JwkPublicSources {
	return c.sources
}

func (c *client) Recipients() JwkRecipients {
	return c.recipients
}

func (c *client) Keys() map[string]*JwkConfig {
	return c.keys
}

func (c *client) Close() {
	_ = c.conn.Close()
}

// NewClient dials the JSON-keys gRPC server at addr and prepares the client for use,
// returning an error if any setup step fails. Any grpc.DialOption values accepted by
// grpc.NewClient may be forwarded via opts.
func NewClient(addr string, opts ...grpc.DialOption) (Client, error) {
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("new grpc client: %w", err)
	}

	c := &client{
		EchoServiceClient:       golibproto.NewEchoServiceClient(conn),
		StatusServiceClient:     protogen.NewStatusServiceClient(conn),
		JwkGetServiceClient:     protogen.NewJwkGetServiceClient(conn),
		JwkListServiceClient:    protogen.NewJwkListServiceClient(conn),
		ClaimsSignServiceClient: protogen.NewClaimsSignServiceClient(conn),
		keys:                    config.JwkPresetDefault,
		conn:                    conn,
	}

	adapter := newJwkExportGrpc(c)

	sources, err := services.NewJwkPublicSource(adapter, config.JwkPresetDefault)
	if err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("new jwk public source: %w", err)
	}

	c.sources = sources

	recipients, err := services.NewJwkRecipients(sources, config.JwkPresetDefault)
	if err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("new jwk recipients: %w", err)
	}

	c.recipients = recipients

	return c, nil
}
