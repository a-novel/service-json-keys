package handlers

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// GrpcClaimsSignService is the service dependency of [GrpcClaimsSign].
type GrpcClaimsSignService interface {
	Exec(ctx context.Context, request *services.ClaimsSignRequest) (string, error)
}

// GrpcClaimsSign is the gRPC handler that signs a set of claims and returns a compact JWT.
type GrpcClaimsSign struct {
	protogen.UnimplementedClaimsSignServiceServer

	service GrpcClaimsSignService
}

// NewGrpcClaimsSign returns a new GrpcClaimsSign handler backed by the given service.
func NewGrpcClaimsSign(service GrpcClaimsSignService) *GrpcClaimsSign {
	return &GrpcClaimsSign{service: service}
}

func (handler *GrpcClaimsSign) ClaimsSign(
	ctx context.Context, request *protogen.ClaimsSignRequest,
) (*protogen.ClaimsSignResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.ClaimsSign")
	defer span.End()

	extractedClaims, err := grpcf.ProtoAnyToInterface(request.GetPayload())
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.InvalidArgument, "invalid payload")
	}

	signed, err := handler.service.Exec(ctx, &services.ClaimsSignRequest{
		Claims: extractedClaims,
		Usage:  request.GetUsage(),
	})
	if errors.Is(err, services.ErrConfigNotFound) {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Unavailable, "unknown usage")
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &protogen.ClaimsSignResponse{Token: signed}, nil
}
