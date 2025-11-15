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

//go:embed pg.jwkDelete.sql
var jwkDeleteQuery string

var ErrJwkDeleteNotFound = errors.New("key not found")

type JwkDeleteRequest struct {
	// ID of the key to delete.
	ID uuid.UUID
	// Time at which the key will be marked as deleted.
	Now time.Time
	// Comment gives information about why a key was deleted.
	Comment string
}

// JwkDelete prematurely deletes a JSON web key. It performs a soft delete, meaning the key disappears from the API
// results but remains available for admins.
//
// This repository SHOULD NOT be called when a key expires, as it will naturally be removed from the view anyway.
//
// This method only targets active keys. If the key is already deleted or expired, it will throw with
// ErrJwkDeleteNotFound.
type JwkDelete struct{}

func NewJwkDelete() *JwkDelete {
	return new(JwkDelete)
}

func (repository *JwkDelete) Exec(ctx context.Context, request *JwkDeleteRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.JwkDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", request.ID.String()),
		attribute.Int64("key.expires_at", request.Now.Unix()),
		attribute.String("key.comment", request.Comment),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Jwk)

	err = tx.NewRaw(jwkDeleteQuery, request.Now, request.Comment, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrJwkDeleteNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
