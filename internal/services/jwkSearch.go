package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/dao"
)

type JwkSearchRepository interface {
	Exec(ctx context.Context, request *dao.JwkSearchRequest) ([]*dao.Jwk, error)
}

type JwkSearchServiceExtract interface {
	Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error)
}

type JwkSearchRequest struct {
	Usage   string
	Private bool
}

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
