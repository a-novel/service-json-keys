package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Jwk represents a JSON web key stored in the database. Keys are stored as base64 raw URL encoded strings.
//
// Private keys must be encoded using the environment master key.
//
// A Jwk is never hard deleted. Instead, the repositories interact through an "active_keys" view,
// that contains all keys that have not expired / be deleted yet.
//
// Keys are grouped by Usage (see more in the field documentation). For each usage, there is
// a single main key (the latest), and a bunch of legacy keys (previous owners of the main
// title). All those keys are part of the active_keys view, as long as they meet the conditions
// (not expired nor deleted).
type Jwk struct {
	bun.BaseModel `bun:"table:keys,select:active_keys"`

	// ID of the key. Should match the "kid" parameter when working with JSON web tokens.
	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// PrivateKey is the private key in JSON Web Key format.
	//
	// This key MUST BE encrypted, and the result of this encryption is stored as a base64 raw URL encoded string.
	PrivateKey string `bun:"private_key"`
	// PublicKey is the public key in JSON Web Key format. The key is stored as a base64 raw URL encoded string.
	//
	// This value is OPTIONAL for symmetric keys.
	PublicKey *string `bun:"public_key"`

	// Usage gives information about the operation this key is intended for.
	//
	// A particular Usage value should be registered by a single service, which becomes
	// the "producer" for this usage. Only the producer should be allowed to perform
	// operations requiring the private key (e.g. generating a token).
	//
	// Any client may consume the public key for a given usage, and become a "recipient".
	// Recipients can use the public key of a producer to verify the data they receive.
	//
	// All keys registered under the same usage are considered different versions of the
	// same key. When a new key is registered for a usage, it becomes the "main" key, and
	// the other keys are converted to "legacy" keys.
	//
	// A producer should only ever use the main key for its operations. Legacy keys are
	// only used by recipients to validate / decrypt older data from the producer.
	Usage string `bun:"usage"`

	CreatedAt time.Time `bun:"created_at"`
	// All keys are required to expire. Once the ExpiresAt date is passed, the key
	// is removed from the active keys view, becoming accessible only to admins.
	ExpiresAt time.Time `bun:"expires_at"`

	// DeletedAt indicates the key was deleted prematurely, for example due to a leakage.
	// More information about this deletion may be found in the DeletedComment field.
	//
	// A key that expires naturally does not have a DeletedAt field.
	DeletedAt *time.Time `bun:"deleted_at"`
	// DeletedComment gives information about the premature deletion of a key.
	//
	// See DeletedAt documentation for more information.
	DeletedComment *string `bun:"deleted_comment"`
}
