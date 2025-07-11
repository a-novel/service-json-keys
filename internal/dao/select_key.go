package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed select_key.sql
var selectKeyQuery string

// SelectKeyRepository is the repository used to perform the SelectKeyRepository.SelectKey action.
//
// You may create one using the NewSelectKeyRepository function.
type SelectKeyRepository struct{}

func NewSelectKeyRepository() *SelectKeyRepository {
	return &SelectKeyRepository{}
}

// SelectKey returns a public/private key pair based on their unique identifier (ID).
//
// The ID of a key pair is usually carried by the payload they were used on, for example thw KIS field of a JWT header.
// This allows to retrieve the exact key when performing reverse operations (signature verification or token
// decryption).
func (repository *SelectKeyRepository) SelectKey(ctx context.Context, id uuid.UUID) (*KeyEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SelectKey")
	defer span.End()

	span.SetAttributes(attribute.String("key.id", id.String()))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	var entity KeyEntity

	err = tx.NewRaw(selectKeyQuery, id).Scan(ctx, &entity)
	if err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, ErrKeyNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select key: %w", err))
	}

	return otel.ReportSuccess(span, &entity), nil
}
