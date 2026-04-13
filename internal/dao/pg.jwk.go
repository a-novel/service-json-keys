package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// A Jwk represents a JSON Web Key stored in the database. Keys are stored as base64 raw URL encoded strings.
//
// Private keys must be encrypted with the master key before storage.
//
// A Jwk is never hard-deleted. Instead, repositories query through an "active_keys" view that
// excludes expired and soft-deleted rows. Keys are grouped by [Jwk.Usage]; for each usage there
// is a single main key (the latest) and zero or more legacy keys (older active versions).
type Jwk struct {
	bun.BaseModel `bun:"table:keys,select:active_keys"`

	// ID is the key's unique identifier; it corresponds to the "kid" field in the JWT header.
	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// PrivateKey is the private key in JSON Web Key format, encrypted with the master key
	// and stored as a base64 raw URL encoded ciphertext.
	PrivateKey string `bun:"private_key"`
	// PublicKey is the public key in JSON Web Key format, stored as a base64 raw URL encoded string.
	// It is nil for symmetric keys, which have no public counterpart.
	PublicKey *string `bun:"public_key"`

	// Usage identifies the signing purpose this key serves.
	//
	// A particular Usage value should be registered by a single service, which becomes
	// the "producer" for this usage. Only the producer should be allowed to perform
	// operations requiring the private key (e.g., generating a token).
	//
	// Any client may consume the public key for a given usage and become a "recipient".
	// Recipients can use the public key of a producer to verify the data they receive.
	//
	// All keys registered under the same usage are considered different versions of the
	// same key. When a new key is registered for a usage, it becomes the "main" key, and
	// the other keys are converted to "legacy" keys.
	//
	// A producer signs only with the main key. Recipients can verify tokens signed
	// by any active key for the usage.
	Usage string `bun:"usage"`

	// CreatedAt is when the key was generated and persisted; it determines the
	// rotation schedule — a new key is created when the main key's age exceeds the
	// configured rotation interval.
	CreatedAt time.Time `bun:"created_at"`
	// ExpiresAt is the hard expiry date. Once passed, the key leaves the active view
	// and is only accessible via direct database queries.
	ExpiresAt time.Time `bun:"expires_at"`

	// DeletedAt is set when the key is revoked prematurely — for example, due to a compromise.
	// It is nil for keys that expire naturally. See [Jwk.DeletedComment] for the reason.
	DeletedAt *time.Time `bun:"deleted_at"`
	// DeletedComment is the human-readable reason for the early revocation. See [Jwk.DeletedAt].
	DeletedComment *string `bun:"deleted_comment"`
}
