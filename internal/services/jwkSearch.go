package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

// JwkSearchRepository is the DAO dependency of [JwkSearch].
type JwkSearchRepository interface {
	Exec(ctx context.Context, request *dao.JwkSearchRequest) ([]*dao.Jwk, error)
}

// JwkSearchServiceExtract is the service dependency of [JwkSearch] for deserializing DAO entities.
type JwkSearchServiceExtract interface {
	Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error)
}

// JwkSearchRequest holds the parameters for a [JwkSearch.Exec] call.
type JwkSearchRequest struct {
	// Usage is the key usage to filter by.
	Usage string
	// Private controls whether to return the private key material. Set to true only for the signing
	// path (gRPC ClaimsSign); public key endpoints must leave this false.
	Private bool
}

// A JwkSearch lists the active keys for a given usage. Keys are returned in
// creation order: the first element is the main key, the rest are legacy.
type JwkSearch struct {
	repository     JwkSearchRepository
	serviceExtract JwkSearchServiceExtract
}

// NewJwkSearch returns a new JwkSearch service.
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
	ctx, span := otel.Tracer().Start(ctx, "services.JwkSearch")
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
	deserialized := make([]*Jwk, len(entities))

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
