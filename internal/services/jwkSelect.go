package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

type JwkSelectRepository interface {
	Exec(ctx context.Context, request *dao.JwkSelectRequest) (*dao.Jwk, error)
}

type JwkSelectServiceExtract interface {
	Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error)
}

type JwkSelectRequest struct {
	// ID of the key to retrieve. This parameter is usually available under the "kid" field
	// of a JSON web token claims / headers.
	ID uuid.UUID
	// Whether to return private or public keys.
	//
	// Note: if this option is set to true, make sure the query comes from the
	// correct producer.
	Private bool
}

// JwkSelect retrieves a key from its ID. The key id is usually available under the "kid" field
// of a JSON web token, but can also appear under different fields.
type JwkSelect struct {
	repository     JwkSelectRepository
	serviceExtract JwkSelectServiceExtract
}

func NewJwkSelect(
	repository JwkSelectRepository,
	serviceExtract JwkSelectServiceExtract,
) *JwkSelect {
	return &JwkSelect{
		repository:     repository,
		serviceExtract: serviceExtract,
	}
}

func (service *JwkSelect) Exec(ctx context.Context, request *JwkSelectRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JwkSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", request.ID.String()),
		attribute.Bool("key.private", request.Private),
	)

	entity, err := service.repository.Exec(ctx, &dao.JwkSelectRequest{
		ID: request.ID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select key: %w", err))
	}

	deserialized, err := service.serviceExtract.Exec(ctx, &JwkExtractRequest{
		Jwk:     entity,
		Private: request.Private,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("deserialize key: %w", err))
	}

	return otel.ReportSuccess(span, deserialized), nil
}
