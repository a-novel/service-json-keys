package config

import "github.com/a-novel/service-json-keys/v2/internal/jwk"

type (
	// JwkKey holds the lifetime and caching parameters for a JSON Web Key.
	JwkKey = jwk.JwkKey
	// JwkToken holds the claims parameters applied to every JWT signed with a given key.
	JwkToken = jwk.JwkToken
	// Jwk holds the full configuration for a single key usage.
	Jwk = jwk.Jwk
)

// JwkPresetDefault is the default JWK configuration for all registered usages.
var JwkPresetDefault = jwk.PresetDefault
