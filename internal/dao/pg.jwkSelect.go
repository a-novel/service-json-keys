package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.jwkSelect.sql
var jwkSelectQuery string

// ErrJwkSelectNotFound is returned when no active key matches the requested ID.
var ErrJwkSelectNotFound = errors.New("jwk not found")

// JwkSelectRequest holds the parameters for a [JwkSelect.Exec] call.
type JwkSelectRequest struct {
	// ID is the key to retrieve; it corresponds to the "kid" field in the JWT header.
	ID uuid.UUID
}

// A JwkSelect retrieves a single active key by its ID.
type JwkSelect struct{}

// NewJwkSelect returns a new JwkSelect repository.
func NewJwkSelect() *JwkSelect {
	return &JwkSelect{}
}

func (repository *JwkSelect) Exec(ctx context.Context, request *JwkSelectRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SelectKey")
	defer span.End()

	span.SetAttributes(attribute.String("key.id", request.ID.String()))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	var entity Jwk

	err = tx.NewRaw(jwkSelectQuery, request.ID).Scan(ctx, &entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrJwkSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, &entity), nil
}
