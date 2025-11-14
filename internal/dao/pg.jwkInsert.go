package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.jwkInsert.sql
var jwkInsertQuery string

type JwkInsertRequest struct {
	ID uuid.UUID

	// The private key in JSON Web Key format.
	//
	// The key MUST BE encrypted, and the result of this encryption is stored as a base64 raw URL encoded string.
	PrivateKey string
	// The public key in JSON Web Key format. The key is stored as a base64 raw URL encoded string.
	//
	// This value is OPTIONAL for symmetric keys.
	PublicKey *string

	Usage string

	Now time.Time
	// Expiration of the key. Each key pair is REQUIRED to expire past a certain time. Once the expiration date
	// is reached, the key pair becomes invisible to the keys view.
	Expiration time.Time
}

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

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := &Jwk{
		ID:         request.ID,
		PrivateKey: request.PrivateKey,
		PublicKey:  request.PublicKey,
		Usage:      request.Usage,
		CreatedAt:  request.Now,
		ExpiresAt:  request.Expiration,
	}

	// Execute query.
	err = tx.
		NewRaw(
			jwkInsertQuery,
			entity.ID,
			entity.PrivateKey,
			entity.PublicKey,
			entity.Usage,
			entity.CreatedAt,
			entity.ExpiresAt,
		).
		Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, entity), nil
}
