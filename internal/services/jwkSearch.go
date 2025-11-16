package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

type JwkSearchRepository interface {
	Exec(ctx context.Context, request *dao.JwkSearchRequest) ([]*dao.Jwk, error)
}

type JwkSearchServiceExtract interface {
	Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error)
}

type JwkSearchRequest struct {
	// The intended usage of the key.
	Usage string
	// Whether to return private or public keys.
	//
	// Note: if this option is set to true, make sure the query comes from the
	// correct producer.
	Private bool
}

// JwkSearch lists the active keys for a given usage. The keys are returned in
// creation order (first key in the array is the main key, the rest are legacy).
type JwkSearch struct {
	repository     JwkSearchRepository
	serviceExtract JwkSearchServiceExtract
}

func NewJwkSearch(
	repository JwkSearchRepository,
	serviceExtract JwkSearchServiceExtract,
) *JwkSearch {
	return &JwkSearch{
		repository:     repository,
		serviceExtract: serviceExtract,
	}
}

func (service *JwkSearch) Exec(ctx context.Context, request *JwkSearchRequest) ([]*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JwkSearch")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.usage", request.Usage),
		attribute.Bool("key.private", request.Private),
	)

	entities, err := service.repository.Exec(ctx, &dao.JwkSearchRequest{
		Usage: request.Usage,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("search entities: %w", err))
	}

	span.SetAttributes(attribute.Int("entities.count", len(entities)))

	// Don't use lo so we can handle errors properly.
	deserialized := make([]*jwa.JWK, len(entities))

	for i, entity := range entities {
		deserialized[i], err = service.serviceExtract.Exec(ctx, &JwkExtractRequest{
			Jwk:     entity,
			Private: request.Private,
		})
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("consume DAO entity (kid %s): %w", entity.ID, err))
		}
	}

	return otel.ReportSuccess(span, deserialized), nil
}
