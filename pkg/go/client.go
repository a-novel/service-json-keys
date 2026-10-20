package servicejsonkeys

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

type (
	// StatusRequest is the request of [BaseClient.Status]. It carries no parameters.
	StatusRequest = jsonkeysv2.StatusRequest
	// StatusResponse reports the health of each dependency [BaseClient.Status] checked.
	StatusResponse = jsonkeysv2.StatusResponse
	// JwkListRequest names the usage whose public keys [BaseClient.JwkList] returns.
	JwkListRequest = jsonkeysv2.JwkListRequest
	// JwkListResponse holds the active public keys of a usage, newest first.
	JwkListResponse = jsonkeysv2.JwkListResponse
	// JwkGetRequest names the key ID [BaseClient.JwkGet] looks up.
	JwkGetRequest = jsonkeysv2.JwkGetRequest
	// JwkGetResponse holds the public key [BaseClient.JwkGet] found.
	JwkGetResponse = jsonkeysv2.JwkGetResponse
	// ClaimsSignRequest carries the claims [BaseClient.ClaimsSign] signs and the usage that
	// selects the key.
	ClaimsSignRequest = jsonkeysv2.ClaimsSignRequest
	// ClaimsSignResponse holds the compact JWT [BaseClient.ClaimsSign] produced.
	ClaimsSignResponse = jsonkeysv2.ClaimsSignResponse

	// JwkConfig holds the configuration of one key usage: its signing algorithm, and the key and
	// token parameters of every JWT signed under it. [Client.Keys] maps each usage name to one.
	JwkConfig = jwkconfig.Jwk
)

// BaseClient is the minimal gRPC interface for the JSON Keys service. It exposes every RPC
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
	// It returns Unavailable when a dependency probe fails. Set a request deadline.
	Status(ctx context.Context, req *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)

	// JwkGet retrieves a single JSON Web Key by its key ID.
	JwkGet(ctx context.Context, req *JwkGetRequest, opts ...grpc.CallOption) (*JwkGetResponse, error)
	// JwkList returns the active public keys for a given usage.
	JwkList(ctx context.Context, req *JwkListRequest, opts ...grpc.CallOption) (*JwkListResponse, error)
	// ClaimsSign asks the service to sign claims and returns a compact JWT.
	// The request payload must be a protobuf Any wrapping the claims to embed.
	// A usage the service has no configuration for fails with InvalidArgument.
	//
	// The payload carries application claims only. The service stamps the registered
	// ones (iss, sub, aud, exp, nbf, iat, jti) from the usage's server-side config, and
	// a payload naming one fails with InvalidArgument.
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
	jsonkeysv2.StatusServiceClient
	jsonkeysv2.JwkGetServiceClient
	jsonkeysv2.JwkListServiceClient
	jsonkeysv2.ClaimsSignServiceClient

	keys map[string]*JwkConfig

	conn *grpc.ClientConn
}

func (c *client) Keys() map[string]*JwkConfig {
	return c.keys
}

func (c *client) Close() {
	_ = c.conn.Close()
}

// NewClient dials the JSON Keys gRPC server at addr and prepares the client for use,
// returning an error if any setup step fails. Any grpc.DialOption values accepted by
// grpc.NewClient may be forwarded via opts.
//
// Each call opens a client span and sends the caller's trace context, so the server's
// spans join the caller's trace.
func NewClient(addr string, opts ...grpc.DialOption) (Client, error) {
	opts = append([]grpc.DialOption{grpc.WithStatsHandler(otelgrpc.NewClientHandler())}, opts...)

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("new grpc client: %w", err)
	}

	c := &client{
		EchoServiceClient:       golibproto.NewEchoServiceClient(conn),
		StatusServiceClient:     jsonkeysv2.NewStatusServiceClient(conn),
		JwkGetServiceClient:     jsonkeysv2.NewJwkGetServiceClient(conn),
		JwkListServiceClient:    jsonkeysv2.NewJwkListServiceClient(conn),
		ClaimsSignServiceClient: jsonkeysv2.NewClaimsSignServiceClient(conn),
		keys:                    jwkconfig.PresetDefault,
		conn:                    conn,
	}

	return c, nil
}
