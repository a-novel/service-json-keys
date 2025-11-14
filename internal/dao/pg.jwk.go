package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Jwk struct {
	bun.BaseModel `bun:"table:keys,select:active_keys"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// The private key in JSON Web Key format.
	//
	// The key MUST BE encrypted, and the result of this encryption is stored as a base64 raw URL encoded string.
	PrivateKey string `bun:"private_key"`
	// The public key in JSON Web Key format. The key is stored as a base64 raw URL encoded string.
	//
	// This value is OPTIONAL for symmetric keys.
	PublicKey *string `bun:"public_key"`

	// Intended usage of the key. See the type documentation for more details.
	Usage string `bun:"usage"`

	CreatedAt time.Time `bun:"created_at"`
	ExpiresAt time.Time `bun:"expires_at"`

	DeletedAt      *time.Time `bun:"deleted_at"`
	DeletedComment *string    `bun:"deleted_comment"`
}
