package handlers

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel/golib/grpcf"
	"github.com/a-novel/golib/otel"

	protogen "github.com/a-novel/service-json-keys/v2/internal/handlers/proto/gen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

type ClaimsSignService interface {
	Exec(ctx context.Context, request *services.ClaimsSignRequest) (string, error)
}

type ClaimsSign struct {
	protogen.UnimplementedClaimsSignServiceServer

	service ClaimsSignService
}

func NewClaimsSign(service ClaimsSignService) *ClaimsSign {
	return &ClaimsSign{service: service}
}

func (handler *ClaimsSign) ClaimsSign(
	ctx context.Context, request *protogen.ClaimsSignRequest,
) (*protogen.ClaimsSignResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "handler.ClaimsSign")
	defer span.End()

	extractedClaims, err := grpcf.ProtoAnyToInterface(request.GetPayload())
	if err != nil {
		return nil, otel.ReportError(span, status.Error(codes.InvalidArgument, err.Error()))
	}

	signed, err := handler.service.Exec(ctx, &services.ClaimsSignRequest{
		Claims: extractedClaims,
		Usage:  request.GetUsage(),
	})
	if errors.Is(err, services.ErrConfigNotFound) {
		return nil, otel.ReportError(span, status.Error(codes.Unavailable, err.Error()))
	}

	if err != nil {
		return nil, otel.ReportError(span, status.Error(codes.Internal, err.Error()))
	}

	return &protogen.ClaimsSignResponse{Token: signed}, nil
}
