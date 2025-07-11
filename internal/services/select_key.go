package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/dao"
)

// SelectKeySource is the source used to perform the SelectKeyService.SelectKey action.
type SelectKeySource interface {
	SelectKey(ctx context.Context, id uuid.UUID) (*dao.KeyEntity, error)
}

// SelectKeyRequest is the input used to perform the SelectKeyService.SelectKey action.
type SelectKeyRequest struct {
	// ID of the target key. This matches the "kid" header.
	ID uuid.UUID
	// If true, return the private key. Otherwise, return the public key.
	Private bool
}

// SelectKeyService is the service used to perform the SelectKeyService.SelectKey action.
//
// You may create one using the NewSelectKeyService function.
type SelectKeyService struct {
	source SelectKeySource
}

func NewSelectKeyService(source SelectKeySource) *SelectKeyService {
	return &SelectKeyService{source: source}
}

func (service *SelectKeyService) SelectKey(ctx context.Context, request SelectKeyRequest) (*jwa.JWK, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SelectKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", request.ID.String()),
		attribute.Bool("key.private", request.Private),
	)

	key, err := service.source.SelectKey(ctx, request.ID)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select key: %w", err))
	}

	deserialized, err := dao.ConsumeKey(ctx, key, request.Private)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("consume DAO key (kid %s): %w", key.ID, err))
	}

	return otel.ReportSuccess(span, deserialized), nil
}
