package handlers

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

// GrpcAPIKeyGetService is the service dependency of [GrpcAPIKeyGet].
type GrpcAPIKeyGetService interface {
	Exec(ctx context.Context, request *core.APIKeySelectRequest) (*core.APIKey, error)
}

// GrpcAPIKeyGet is the gRPC handler that retrieves an API key by its ID.
type GrpcAPIKeyGet struct {
	jsonkeysv2.UnimplementedAPIKeyGetServiceServer

	service GrpcAPIKeyGetService
}

func NewGrpcAPIKeyGet(service GrpcAPIKeyGetService) *GrpcAPIKeyGet {
	return &GrpcAPIKeyGet{service: service}
}

func (handler *GrpcAPIKeyGet) APIKeyGet(
	ctx context.Context, request *jsonkeysv2.APIKeyGetRequest,
) (*jsonkeysv2.APIKeyGetResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.APIKeyGet")
	defer span.End()

	key, err := handler.service.Exec(ctx, &core.APIKeySelectRequest{ID: request.GetId()})
	if errors.Is(err, core.ErrAPIKeyNotFound) {
		return nil, status.Error(codes.NotFound, "api key not found")
	}

	if err != nil {
		return nil, grpcf.Fault(codes.Internal, "internal error", otel.ReportError(span, err))
	}

	return &jsonkeysv2.APIKeyGetResponse{ApiKey: &jsonkeysv2.APIKey{
		Id:          key.ID,
		Purpose:     key.Purpose,
		Jwk:         jwkToProto(key.Key),
		Deactivated: key.Deactivated,
		Leaked:      key.Leaked,
	}}, nil
}
