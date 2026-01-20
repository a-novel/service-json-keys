package handlers

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

type JwkGetService interface {
	Exec(ctx context.Context, request *services.JwkSelectRequest) (*services.Jwk, error)
}

type JwkGet struct {
	protogen.UnimplementedJwkGetServiceServer

	service JwkGetService
}

func NewJwkGet(service JwkGetService) *JwkGet {
	return &JwkGet{service: service}
}

func (handler *JwkGet) JwkGet(ctx context.Context, request *protogen.JwkGetRequest) (*protogen.JwkGetResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "handler.JwkGet")
	defer span.End()

	keyId, err := uuid.Parse(request.GetId())
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid key id")
	}

	jwk, err := handler.service.Exec(ctx, &services.JwkSelectRequest{
		ID: keyId,
	})
	if errors.Is(err, dao.ErrJwkSelectNotFound) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.NotFound, "key not found")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &protogen.JwkGetResponse{
		Jwk: &protogen.Jwk{
			Kty:     jwk.KTY.String(),
			Use:     jwk.Use.String(),
			KeyOps:  jwk.KeyOps.Strings(),
			Alg:     jwk.Alg.String(),
			Kid:     jwk.KID,
			Payload: jwk.Payload,
		},
	}, nil
}
