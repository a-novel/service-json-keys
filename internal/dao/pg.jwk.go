package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// A Jwk is a JSON Web Key row. Both halves are stored as base64 raw URL encoded strings, and the
// private half is encrypted with the master key.
//
// Rows stay in the table once they expire or are revoked. DAOs read through the active_keys view,
// which hides those rows.
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
	// A usage is registered by a single service, its "producer", and only the producer may use
	// the private key to sign. Any client may fetch the public key for a usage and become a
	// "recipient" that verifies what the producer signed.
	//
	// All keys under one usage are versions of the same key: the newest is the "main" key, the
	// rest are "legacy". A producer signs only with the main key; recipients accept any active
	// key for the usage.
	Usage string `bun:"usage"`

	// CreatedAt is when the key was generated and persisted. A new key is generated once the
	// main key is older than the configured rotation interval.
	CreatedAt time.Time `bun:"created_at"`
	// ExpiresAt is the hard expiry date. Once passed, the key leaves the active view
	// and is only accessible via direct database queries.
	ExpiresAt time.Time `bun:"expires_at"`

	// DeletedAt is set when the key is revoked before it expires, after a compromise for example.
	// It is nil for keys that expire naturally. See [Jwk.DeletedComment] for the reason.
	DeletedAt *time.Time `bun:"deleted_at"`
	// DeletedComment is the human-readable reason for the early revocation. See [Jwk.DeletedAt].
	DeletedComment *string `bun:"deleted_comment"`
}
