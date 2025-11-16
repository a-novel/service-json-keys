package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/jwt/jwa"
)

type JwkKey struct {
	// TTL configures how long a key version will remain available once created.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	// Rotation configures the interval at which new versions of the key are created.
	// It should be significantly lower than the TTL.
	Rotation time.Duration `json:"rotation" yaml:"rotation"`
	// Cache configures how long a key should be cached in memory, before its value
	// is refetched from the database.
	// It should be significantly lower than the TTL.
	Cache time.Duration `json:"cache" yaml:"cache"`
}

type JwkToken struct {
	// TTL configures how long a new token will remain valid.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	// Issuer of the token, as it will appear in the claims.
	Issuer string `json:"issuer" yaml:"issuer"`
	// Audience for the token, as it will appear in the claims.
	Audience string `json:"audience" yaml:"audience"`
	// Subject of the token, as it will appear in the claims.
	Subject string `json:"subject" yaml:"subject"`
	// Leeway allowed when checking for token validity.
	Leeway time.Duration `json:"leeway" yaml:"leeway"`
}

// Jwk configuration for a single usage.
type Jwk struct {
	// Alg the keys will be used for.
	Alg jwa.Alg `json:"alg" yaml:"alg"`
	// Configuration parameters for Json Web Keys (JWK).
	Key JwkKey `json:"key" yaml:"key"`
	// Configuration parameters for Json Web Token (JWT).
	Token JwkToken `json:"token" yaml:"token"`
}
