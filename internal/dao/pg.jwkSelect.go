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
	// ID of the key to retrieve. This parameter is usually available under the "kid" field
	// of a JSON web token claims / headers.
	ID uuid.UUID
}

// JwkSelect retrieves a key from its ID. The key id is usually available under the "kid" field
// of a JSON web token, but can also appear under different fields.
type JwkSelect struct{}

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
