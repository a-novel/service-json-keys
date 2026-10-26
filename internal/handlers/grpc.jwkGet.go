package handlers

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

// GrpcJwkGetService is the service dependency of [GrpcJwkGet].
type GrpcJwkGetService interface {
	Exec(ctx context.Context, request *core.JwkSelectRequest) (*core.Jwk, error)
}

// GrpcJwkGet is the gRPC handler that retrieves a single JSON Web Key by its ID.
type GrpcJwkGet struct {
	jsonkeysv2.UnimplementedJwkGetServiceServer

	service GrpcJwkGetService
}

func NewGrpcJwkGet(service GrpcJwkGetService) *GrpcJwkGet {
	return &GrpcJwkGet{service: service}
}

func (handler *GrpcJwkGet) JwkGet(
	ctx context.Context, request *jsonkeysv2.JwkGetRequest,
) (*jsonkeysv2.JwkGetResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.JwkGet")
	defer span.End()

	keyID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid key id")
	}

	jwk, err := handler.service.Exec(ctx, &core.JwkSelectRequest{ID: keyID})
	if errors.Is(err, core.ErrJwkNotFound) {
		return nil, status.Error(codes.NotFound, "jwk not found")
	}

	if err != nil {
		return nil, grpcf.Fault(codes.Internal, "internal error", otel.ReportError(span, err))
	}

	return &jsonkeysv2.JwkGetResponse{Jwk: jwkToProto(jwk)}, nil
}

// jwkToProto converts a core key into its protobuf form.
func jwkToProto(jwk *core.Jwk) *jsonkeysv2.Jwk {
	return &jsonkeysv2.Jwk{
		Kty:     jwk.KTY.String(),
		Use:     jwk.Use.String(),
		KeyOps:  jwk.KeyOps.Strings(),
		Alg:     jwk.Alg.String(),
		Kid:     jwk.KID,
		Payload: jwk.Payload,
	}
}
