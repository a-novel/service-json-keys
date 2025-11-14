package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.jwkSearch.sql
var jwkSearchQuery string

// KeysMaxBatchSize is a security used to limit the number of keys retrieved by a search operation.
//
// Normally, regular key rotation and well configured expiration should limit the number of keys per batch, so the
// search request has no pagination. Since we can't overrule an issue that would cause the number of keys in a batch
// to balloon, this value is used as a security measurement, to guarantee an upper limit of keys retrieved.
const KeysMaxBatchSize = 100

var ErrJwkSearchTooManyResults = fmt.Errorf("more than %d keys found", KeysMaxBatchSize)

type JwkSearchRequest struct {
	Usage string
}

type JwkSearch struct{}

func NewJwkSearch() *JwkSearch {
	return new(JwkSearch)
}

func (repository *JwkSearch) Exec(ctx context.Context, request *JwkSearchRequest) ([]*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.JwkSearch")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var entities []*Jwk

	// Adda +1 to the limit, so we can differentiate between limit reached (which is OK) and limit exceeded
	// (which is not).
	err = tx.NewRaw(jwkSearchQuery, request.Usage, KeysMaxBatchSize+1).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list keys: %w", err))
	}

	span.SetAttributes(
		attribute.Int("keys.count", len(entities)),
		attribute.Int("keys.max_batch_size", KeysMaxBatchSize),
	)

	// Log an error when too many keys are found. This indicates a potential misconfiguration.
	if len(entities) > KeysMaxBatchSize {
		err = fmt.Errorf("%w: %d keys found for usage %s", ErrJwkSearchTooManyResults, len(entities), request.Usage)

		logger := otel.Logger()
		logger.ErrorContext(ctx, err.Error())
		span.RecordError(err)

		// Truncate the list to the maximum allowed size.
		entities = entities[:KeysMaxBatchSize]
	}

	return otel.ReportSuccess(span, entities), nil
}
