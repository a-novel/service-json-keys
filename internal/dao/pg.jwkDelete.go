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

var ErrJwkDeleteNotFound = errors.New("jwk delete not found")

type JwkDeleteRequest struct {
	ID      uuid.UUID
	Now     time.Time
	Comment string
}

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

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := &Jwk{
		ID:             request.ID,
		DeletedAt:      &request.Now,
		DeletedComment: &request.Comment,
	}

	// Execute query.
	err = tx.NewRaw(jwkDeleteQuery, entity.DeletedAt, entity.DeletedComment, entity.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, errors.Join(err, ErrJwkDeleteNotFound))
		}

		return nil, otel.ReportError(span, fmt.Errorf("delete key: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
