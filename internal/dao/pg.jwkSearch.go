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
// to balloon, this value is used as a security measure, to guarantee an upper limit of keys retrieved.
const KeysMaxBatchSize = 100

var ErrJwkSearchTooManyResults = fmt.Errorf("the query returned %d or more results", KeysMaxBatchSize)

type JwkSearchRequest struct {
	// See Jwk.Usage.
	Usage string
}

// JwkSearch lists the active keys for a given usage. The keys are returned in
// creation order (first key in the array is the main key, the rest are legacy).
//
// There is no pagination for this query, as the number of active keys is guaranteed
// to be lower than KeysMaxBatchSize at any given time.
type JwkSearch struct{}

func NewJwkSearch() *JwkSearch {
	return new(JwkSearch)
}

func (repository *JwkSearch) Exec(ctx context.Context, request *JwkSearchRequest) ([]*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.JwkSearch")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var entities []*Jwk

	err = tx.NewRaw(jwkSearchQuery, request.Usage, KeysMaxBatchSize).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(
		attribute.Int("keys.count", len(entities)),
		attribute.Int("keys.max_batch_size", KeysMaxBatchSize),
	)

	// Log an error when too many keys are found. This indicates a potential misconfiguration.
	// This issue should not be blocking, as we still retrieve the main key.
	if len(entities) >= KeysMaxBatchSize {
		err = fmt.Errorf("%w: %d keys found for usage %s", ErrJwkSearchTooManyResults, len(entities), request.Usage)

		logger := otel.Logger()
		logger.ErrorContext(ctx, err.Error())
		span.RecordError(err)
	}

	return otel.ReportSuccess(span, entities), nil
}
