package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.jwkInsert.sql
var jwkInsertQuery string

type JwkInsertRequest struct {
	// ID of the key to insert. Must be unique among all existing keys.
	ID uuid.UUID

	// See Jwk.PrivateKey. Must be encrypted.
	PrivateKey string
	// See Jwk.PublicKey.
	PublicKey *string
	// See Jwk.Usage.
	Usage string

	// Time used for key creation.
	Now time.Time
	// See Jwk.ExpiresAt.
	Expiration time.Time
}

// JwkInsert inserts a new key for a given usage. If the creation time is greater
// than any existing key for this usage, the new key will become the "main" key.
type JwkInsert struct{}

func NewJwkInsert() *JwkInsert {
	return new(JwkInsert)
}

func (repository *JwkInsert) Exec(ctx context.Context, request *JwkInsertRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.JwkInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", request.ID.String()),
		attribute.String("key.usage", request.Usage),
		attribute.Int64("key.created_at", request.Now.Unix()),
		attribute.Int64("key.expires_at", request.Expiration.Unix()),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Jwk)

	err = tx.
		NewRaw(
			jwkInsertQuery,
			request.ID,
			request.PrivateKey,
			request.PublicKey,
			request.Usage,
			request.Now,
			request.Expiration,
		).
		Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
