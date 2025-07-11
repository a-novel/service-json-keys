package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/services"
)

type SelectKeyService interface {
	SelectKey(ctx context.Context, request services.SelectKeyRequest) (*jwa.JWK, error)
}

func (api *API) GetPublicKey(ctx context.Context, params codegen.GetPublicKeyParams) (codegen.GetPublicKeyRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.GetPublicKey")
	defer span.End()

	key, err := api.SelectKeyService.SelectKey(ctx, services.SelectKeyRequest{
		ID:      uuid.UUID(params.Kid),
		Private: false,
	})

	switch {
	case errors.Is(err, dao.ErrKeyNotFound):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &codegen.NotFoundError{Error: "key not found"}, nil
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return nil, fmt.Errorf("select key: %w", err)
	}

	resp, err := api.jwkToModel(key)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("convert key to model: %w", err))
	}

	return otel.ReportSuccess(span, resp), nil
}
