package handlers

import (
	"context"

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// GrpcJwkListService is the service dependency of [GrpcJwkList].
type GrpcJwkListService interface {
	Exec(ctx context.Context, request *services.JwkSearchRequest) ([]*services.Jwk, error)
}

// GrpcJwkList is the gRPC handler that returns the active public keys for a given usage.
type GrpcJwkList struct {
	protogen.UnimplementedJwkListServiceServer

	service GrpcJwkListService
}

// NewGrpcJwkList returns a new GrpcJwkList handler backed by the given service.
func NewGrpcJwkList(service GrpcJwkListService) *GrpcJwkList {
	return &GrpcJwkList{service: service}
}

func (handler *GrpcJwkList) JwkList(
	ctx context.Context, request *protogen.JwkListRequest,
) (*protogen.JwkListResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.JwkList")
	defer span.End()

	jwks, err := handler.service.Exec(ctx, &services.JwkSearchRequest{
		Usage: request.GetUsage(),
	})
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return otel.ReportSuccess(span, &protogen.JwkListResponse{
		Keys: lo.Map(jwks, func(item *services.Jwk, index int) *protogen.Jwk {
			return &protogen.Jwk{
				Kty:     item.KTY.String(),
				Use:     item.Use.String(),
				KeyOps:  item.KeyOps.Strings(),
				Alg:     item.Alg.String(),
				Kid:     item.KID,
				Payload: item.Payload,
			}
		}),
	}), nil
}
