package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.jwkSearch.sql
var jwkSearchQuery string

// KeysMaxBatchSize is the maximum number of keys returned by a single search operation.
//
// Regular key rotation and TTL-based expiration normally keep the number of active keys per usage
// low, so the search has no pagination. This limit exists as a safeguard in case of misconfiguration
// or unexpected key accumulation.
const KeysMaxBatchSize = 100

// ErrJwkSearchTooManyResults is logged (not returned) when a search hits the [KeysMaxBatchSize]
// limit. Results are still returned, but the condition indicates a likely misconfiguration.
var ErrJwkSearchTooManyResults = fmt.Errorf("the query returned %d or more results", KeysMaxBatchSize)

// JwkSearchRequest holds the parameters for a [PgJwkSearch.Exec] call.
type JwkSearchRequest struct {
	// Usage is the key usage to filter by. See [Jwk.Usage].
	Usage string
}

// A PgJwkSearch lists the active keys for a given usage. Keys are returned in
// creation order: the first element is the main key, the rest are legacy.
//
// There is no pagination for this query; regular rotation and expiration keep
// the active key count well below [KeysMaxBatchSize] under normal operation.
type PgJwkSearch struct{}

// NewPgJwkSearch returns a new PgJwkSearch repository.
func NewPgJwkSearch() *PgJwkSearch {
	return new(PgJwkSearch)
}

func (repository *PgJwkSearch) Exec(ctx context.Context, request *JwkSearchRequest) ([]*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgJwkSearch")
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

	// Log an error when too many keys are found, as this indicates a potential misconfiguration.
	// The results found so far are still returned to the caller.
	if len(entities) >= KeysMaxBatchSize {
		err = fmt.Errorf("%w: %d keys found for usage %s", ErrJwkSearchTooManyResults, len(entities), request.Usage)

		logger := otel.Logger()
		logger.ErrorContext(ctx, err.Error())
		span.RecordError(err)
	}

	return otel.ReportSuccess(span, entities), nil
}
