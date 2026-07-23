package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/jwt/v2/jwa"
)

// JwkKey holds the lifetime and caching parameters for a JSON Web Key.
type JwkKey struct {
	// TTL configures how long a key version remains available after creation.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	// Rotation configures the interval at which new versions of the key are created.
	// It should be significantly lower than the TTL.
	Rotation time.Duration `json:"rotation" yaml:"rotation"`
	// Cache configures how long a key is cached in memory before being refetched from the database.
	// It should be significantly lower than the TTL.
	Cache time.Duration `json:"cache" yaml:"cache"`
	// UnknownKeyIDInterval bounds how often a verifier may refetch when it meets a key id its cache
	// does not hold — the shape a just-rotated key takes before Cache expires. It caps a caller
	// sending unknown ids to one fetch per interval. Zero uses the jwt default.
	UnknownKeyIDInterval time.Duration `json:"unknownKeyIDInterval" yaml:"unknownKeyIDInterval"`
}

// JwkToken holds the claims parameters applied to every JWT signed with a given key.
type JwkToken struct {
	// TTL configures how long a new token remains valid.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	// Issuer is the token issuer embedded in the "iss" claim.
	Issuer string `json:"issuer" yaml:"issuer"`
	// Audience is the token audience embedded in the "aud" claim.
	Audience string `json:"audience" yaml:"audience"`
	// Subject is the token subject embedded in the "sub" claim.
	Subject string `json:"subject" yaml:"subject"`
	// Leeway is the allowed clock skew when checking token validity.
	Leeway time.Duration `json:"leeway" yaml:"leeway"`
}

// Jwk holds the full configuration for a single key usage.
type Jwk struct {
	// Alg is the signing algorithm for keys under this usage.
	Alg jwa.Alg `json:"alg" yaml:"alg"`
	// Key holds the lifetime and caching parameters for the JSON Web Key.
	Key JwkKey `json:"key" yaml:"key"`
	// Token holds the claims parameters applied to every JWT signed with this key.
	Token JwkToken `json:"token" yaml:"token"`
}
