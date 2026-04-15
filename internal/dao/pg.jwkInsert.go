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

// JwkInsertRequest holds the parameters for a [PgJwkInsert.Exec] call.
type JwkInsertRequest struct {
	// ID is the key's unique identifier; it must not collide with any existing key.
	ID uuid.UUID

	// PrivateKey is the encrypted ciphertext of the private key. See [Jwk.PrivateKey].
	PrivateKey string
	// PublicKey is the serialized public key; nil for symmetric algorithms. See [Jwk.PublicKey].
	PublicKey *string
	// Usage is the key's intended signing purpose. See [Jwk.Usage].
	Usage string

	// Now is the timestamp recorded as the key's creation time.
	Now time.Time
	// Expiration is the hard expiry date for this key. See [Jwk.ExpiresAt].
	Expiration time.Time
}

// A PgJwkInsert inserts a new key for a given usage. If the creation time is greater
// than any existing key for this usage, the new key becomes the main key.
type PgJwkInsert struct{}

// NewPgJwkInsert returns a new PgJwkInsert repository.
func NewPgJwkInsert() *PgJwkInsert {
	return new(PgJwkInsert)
}

func (repository *PgJwkInsert) Exec(ctx context.Context, request *JwkInsertRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgJwkInsert")
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
