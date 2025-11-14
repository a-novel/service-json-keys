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

//go:embed pg.jwkSelect.sql
var jwkSelectQuery string

var ErrJwkSelectNotFound = errors.New("jwk not found")

type JwkSelectRequest struct {
	ID uuid.UUID
}

type JwkSelect struct{}

func NewJwkSelect() *JwkSelect {
	return &JwkSelect{}
}

func (repository *JwkSelect) Exec(ctx context.Context, request *JwkSelectRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SelectKey")
	defer span.End()

	span.SetAttributes(attribute.String("key.id", request.ID.String()))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var entity Jwk

	err = tx.NewRaw(jwkSelectQuery, request.ID).Scan(ctx, &entity)
	if err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, errors.Join(err, ErrJwkSelectNotFound))
		}

		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &entity), nil
}
