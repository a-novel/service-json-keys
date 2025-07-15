package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed delete_key.sql
var deleteKeyQuery string

// DeleteKeyData is the input used to perform the DeleteKeyRepository.DeleteKey action.
type DeleteKeyData struct {
	// ID of the key to delete.
	ID uuid.UUID
	// Time at which the key is marked as deleted. This time might be set in the near future to delay the deletion.
	//
	// Once the date is reached, the key is considered as expired and becomes invisible to the application.
	Now time.Time
	// Comment explaining the circumstances surrounding the deletion of the key.
	Comment string
}

// DeleteKeyRepository is the repository used to perform the DeleteKeyRepository.DeleteKey action.
//
// You may create one using the NewDeleteKeyRepository function.
type DeleteKeyRepository struct{}

func NewDeleteKeyRepository() *DeleteKeyRepository {
	return &DeleteKeyRepository{}
}

// DeleteKey performs a soft delete of a KeyEntity.
//
// A KeyEntity expires naturally through its KeyEntity.ExpiresAt field. However, some circumstances may require a key
// to be invalidated earlier (e.g. a security breach). In such cases, this method can be used.
//
// Once a key is marked as deleted, it is not removed from the database to allow further investigation. It is simply
// removed from the main view, which means the application will not see it anymore.
//
// As this method is not intended to be used "normally", a comment giving more details about the circumstance
// surrounding the deletion is required.
//
// This method also returns an error when the key is not found, so you can be sure something was deleted on success.
// The deleted key is returned on success.
func (repository *DeleteKeyRepository) DeleteKey(ctx context.Context, data DeleteKeyData) (*KeyEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.DeleteKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", data.ID.String()),
		attribute.Int64("key.expires_at", data.Now.Unix()),
		attribute.String("key.comment", data.Comment),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entity := &KeyEntity{
		ID:             data.ID,
		DeletedAt:      &data.Now,
		DeletedComment: &data.Comment,
	}

	// Execute query.
	err = tx.NewRaw(deleteKeyQuery, entity.DeletedAt, entity.DeletedComment, entity.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, fmt.Errorf("delete key: %w", ErrKeyNotFound))
		}

		return nil, otel.ReportError(span, fmt.Errorf("delete key: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
