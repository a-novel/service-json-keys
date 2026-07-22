package servicejsonkeys

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
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

	// JwkConfig holds the full configuration for a single key usage — the signing algorithm
	// and the key and token parameters applied to every JWT signed under it.
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

// Client extends [BaseClient] with the JWK configuration needed to build local token verifiers.
//
// Obtain a Client with [NewClient], then pass it to [NewClaimsVerifier], which builds the
// cached public-key sources used for local verification.
type Client interface {
	BaseClient

	// Keys returns the JWK configuration map this client was built with, keyed by usage name.
	Keys() map[string]*JwkConfig
}

type client struct {
	golibproto.EchoServiceClient
	protogen.StatusServiceClient
	protogen.JwkGetServiceClient
	protogen.JwkListServiceClient
	protogen.ClaimsSignServiceClient

	keys map[string]*JwkConfig

	conn *grpc.ClientConn
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

	return c, nil
}
