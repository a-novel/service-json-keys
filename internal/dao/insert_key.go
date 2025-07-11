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

	"github.com/a-novel/service-json-keys/models"
)

//go:embed insert_key.sql
var insertKeyQuery string

// InsertKeyData is the input used to perform the InsertKeyRepository.InsertKey action.
type InsertKeyData struct {
	// ID of the new key. It MUST be unique (random).
	ID uuid.UUID

	// The private key in JSON Web Key format.
	//
	// The key MUST BE encrypted, and the result of this encryption is stored as a base64 raw URL encoded string.
	PrivateKey string
	// The public key in JSON Web Key format. The key is stored as a base64 raw URL encoded string.
	//
	// This value is OPTIONAL for symmetric keys.
	PublicKey *string

	// Intended usage of the key. See the type documentation for more details.
	Usage models.KeyUsage

	// Time at which the key was created. This is important when listing keys, as the most recent keys are
	// used in priority.
	Now time.Time
	// Expiration of the key. Each key pair is REQUIRED to expire past a certain time. Once the expiration date
	// is reached, the key pair becomes invisible to the keys view.
	Expiration time.Time
}

// InsertKeyRepository is the repository used to perform the InsertKeyRepository.InsertKey action.
//
// You may create one using the NewInsertKeyRepository function.
type InsertKeyRepository struct{}

func NewInsertKeyRepository() *InsertKeyRepository {
	return &InsertKeyRepository{}
}

// InsertKey inserts a new key pair in the database.
//
// A given key pair is REQUIRED to have an expiration date, as it must be rotated on a regular basis. Only public keys
// may be exposed to the application.
func (repository *InsertKeyRepository) InsertKey(ctx context.Context, data InsertKeyData) (*KeyEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.InsertKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("key.id", data.ID.String()),
		attribute.String("key.usage", data.Usage.String()),
		attribute.Int64("key.created_at", data.Now.Unix()),
		attribute.Int64("key.expires_at", data.Expiration.Unix()),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entity := &KeyEntity{
		ID:         data.ID,
		PrivateKey: data.PrivateKey,
		PublicKey:  data.PublicKey,
		Usage:      data.Usage,
		CreatedAt:  data.Now,
		ExpiresAt:  data.Expiration,
	}

	// Execute query.
	err = tx.
		NewRaw(
			insertKeyQuery,
			entity.ID,
			entity.PrivateKey,
			entity.PublicKey,
			entity.Usage,
			entity.CreatedAt,
			entity.ExpiresAt,
		).
		Scan(ctx, entity)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("insert entity: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
