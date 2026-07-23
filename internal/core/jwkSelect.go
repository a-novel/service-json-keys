package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

// JwkSelectDao is the DAO dependency of [JwkSelect].
type JwkSelectDao interface {
	Exec(ctx context.Context, request *dao.JwkSelectRequest) (*dao.Jwk, error)
}

// JwkSelectServiceExtract is the service dependency of [JwkSelect] for deserializing DAO entities.
type JwkSelectServiceExtract interface {
	Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error)
}

// JwkSelectRequest holds the parameters for a [JwkSelect.Exec] call.
type JwkSelectRequest struct {
	// ID is the key to retrieve; it corresponds to the "kid" field in the JWT header.
	ID uuid.UUID
	// Private controls whether to return the private key material. Set to true only for the signing
	// path (gRPC ClaimsSign); public key endpoints must leave this false.
	Private bool
}

// A JwkSelect retrieves a JSON Web Key by its key ID.
type JwkSelect struct {
	dao            JwkSelectDao
	serviceExtract JwkSelectServiceExtract
}

// NewJwkSelect returns a new JwkSelect service.
func NewJwkSelect(
	dao JwkSelectDao,
	serviceExtract JwkSelectServiceExtract,
) *JwkSelect {
	return &JwkSelect{
		dao:            dao,
		serviceExtract: serviceExtract,
	}
}

func (service *JwkSelect) Exec(ctx context.Context, request *JwkSelectRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.JwkSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", request.ID.String()),
		attribute.Bool("key.private", request.Private),
	)

	entity, err := service.dao.Exec(ctx, &dao.JwkSelectRequest{
		ID: request.ID,
	})
	if err != nil {
		if errors.Is(err, dao.ErrJwkSelectNotFound) {
			return nil, otel.ReportError(span, ErrJwkNotFound)
		}

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
