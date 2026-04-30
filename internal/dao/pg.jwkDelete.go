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

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.jwkDelete.sql
var jwkDeleteQuery string

// ErrJwkDeleteNotFound is returned when no active key matches the delete request.
var ErrJwkDeleteNotFound = errors.New("jwk not found")

// JwkDeleteRequest holds the parameters for a [PgJwkDelete.Exec] call.
type JwkDeleteRequest struct {
	// ID is the identifier of the key to revoke.
	ID uuid.UUID
	// Now is the timestamp recorded as the revocation time.
	Now time.Time
	// Comment is the human-readable reason for the revocation, stored for auditing.
	Comment string
}

// A PgJwkDelete prematurely soft-deletes a JSON Web Key: the key is removed from the active
// view and disappears from API results, but is retained in the database for auditing.
//
// Do not call this when a key expires naturally — expired keys are removed from the active view
// automatically. Only active keys can be targeted; if the key is already deleted or expired,
// [ErrJwkDeleteNotFound] is returned.
type PgJwkDelete struct{}

// NewPgJwkDelete returns a new PgJwkDelete repository.
func NewPgJwkDelete() *PgJwkDelete {
	return new(PgJwkDelete)
}

func (repository *PgJwkDelete) Exec(ctx context.Context, request *JwkDeleteRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgJwkDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", request.ID.String()),
		attribute.Int64("key.deleted_at", request.Now.Unix()),
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
			return nil, ErrJwkDeleteNotFound
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
