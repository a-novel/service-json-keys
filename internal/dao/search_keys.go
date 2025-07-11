package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/models"
)

//go:embed search_keys.sql
var searchKeysQuery string

// KeysMaxBatchSize is a security used to limit the number of keys retrieved by a search operation.
//
// Normally, regular key rotation and well configured expiration should limit the number of keys per batch, so the
// search request has no pagination. Since we can't overrule an issue that would cause the number of keys in a batch
// to balloon, this value is used as a security measurement, to guarantee an upper limit of keys retrieved.
const KeysMaxBatchSize = 100

// SearchKeysRepository is the repository used to perform the SearchKeysRepository.SearchKeys action.
//
// You may create one using the NewSearchKeysRepository function.
type SearchKeysRepository struct{}

func NewSearchKeysRepository() *SearchKeysRepository {
	return &SearchKeysRepository{}
}

// SearchKeys lists keys related to a specific usage.
//
// All keys that share the same usage are called a batch. A batch only contains active keys, and is ordered by creation
// date, from the most recent to the oldest. Most recent keys must be used in priority to issue new values. Older keys
// are provided for checking older values only.
//
// While a call to SearchKeys should return every active key without exception, an upper limit is set to prevent a
// potential overhead of the response when too much active keys coexist. This limit is set to KeysMaxBatchSize. If
// a batch happens to contain more keys, an error is logged, and only the first KeysMaxBatchSize keys are returned.
func (repository *SearchKeysRepository) SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*KeyEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SearchKeys")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", usage.String()))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	var entities []*KeyEntity

	// Adda +1 to the limit, so we can differentiate between limit reached (which is OK) and limit exceeded
	// (which is not).
	err = tx.NewRaw(searchKeysQuery, usage, KeysMaxBatchSize+1).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list keys: %w", err))
	}

	span.SetAttributes(
		attribute.Int("keys.count", len(entities)),
		attribute.Int("keys.max_batch_size", KeysMaxBatchSize),
	)

	// Log an error when too many keys are found. This indicates a potential misconfiguration.
	if len(entities) > KeysMaxBatchSize {
		err = fmt.Errorf("more than %d keys found for usage %s", KeysMaxBatchSize, usage)

		logger := otel.Logger()
		logger.ErrorContext(ctx, err.Error())
		span.RecordError(err)

		// Truncate the list to the maximum allowed size.
		entities = entities[:KeysMaxBatchSize]
	}

	return otel.ReportSuccess(span, entities), nil
}
