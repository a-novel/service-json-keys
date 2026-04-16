package handlers

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// GrpcJwkGetService is the service dependency of [GrpcJwkGet].
type GrpcJwkGetService interface {
	Exec(ctx context.Context, request *services.JwkSelectRequest) (*services.Jwk, error)
}

// GrpcJwkGet is the gRPC handler that retrieves a single JSON Web Key by its ID.
type GrpcJwkGet struct {
	protogen.UnimplementedJwkGetServiceServer

	service GrpcJwkGetService
}

// NewGrpcJwkGet returns a new GrpcJwkGet handler backed by the given service.
func NewGrpcJwkGet(service GrpcJwkGetService) *GrpcJwkGet {
	return &GrpcJwkGet{service: service}
}

func (handler *GrpcJwkGet) JwkGet(
	ctx context.Context, request *protogen.JwkGetRequest,
) (*protogen.JwkGetResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.JwkGet")
	defer span.End()

	keyId, err := uuid.Parse(request.GetId())
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid key id")
	}

	jwk, err := handler.service.Exec(ctx, &services.JwkSelectRequest{
		ID: keyId,
	})
	if errors.Is(err, services.ErrJwkNotFound) {
		return nil, status.Error(codes.NotFound, "key not found")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return otel.ReportSuccess(span, &protogen.JwkGetResponse{
		Jwk: &protogen.Jwk{
			Kty:     jwk.KTY.String(),
			Use:     jwk.Use.String(),
			KeyOps:  jwk.KeyOps.Strings(),
			Alg:     jwk.Alg.String(),
			Kid:     jwk.KID,
			Payload: jwk.Payload,
		},
	}), nil
}
