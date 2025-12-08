package pkg

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

	JwkPublicSources = services.JwkPublicSources
	JwkRecipients    = services.JwkRecipients
	JwkConfig        = config.Jwk
)

type BaseClient interface {
	UnaryEcho(
		ctx context.Context, req *golibproto.UnaryEchoRequest, opts ...grpc.CallOption,
	) (*golibproto.UnaryEchoResponse, error)
	Status(ctx context.Context, req *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)

	JwkGet(ctx context.Context, req *JwkGetRequest, opts ...grpc.CallOption) (*JwkGetResponse, error)
	JwkList(ctx context.Context, req *JwkListRequest, opts ...grpc.CallOption) (*JwkListResponse, error)
	ClaimsSign(ctx context.Context, req *ClaimsSignRequest, opts ...grpc.CallOption) (*ClaimsSignResponse, error)

	Close()
}

type Client interface {
	BaseClient

	Sources() *JwkPublicSources
	Recipients() JwkRecipients
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
		return nil, fmt.Errorf("new jwk public source: %w", err)
	}

	c.sources = sources

	recipients, err := services.NewJwkRecipients(sources, config.JwkPresetDefault)
	if err != nil {
		return nil, fmt.Errorf("new jwk recipients: %w", err)
	}

	c.recipients = recipients

	return c, nil
}
