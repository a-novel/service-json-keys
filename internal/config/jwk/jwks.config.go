package jwk

import (
	"errors"
	"time"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
)

// ErrConfigNotFound is returned when no JWK configuration is registered for a given key usage.
var ErrConfigNotFound = errors.New("no config found for the requested usage")

// JwkKey holds the lifetime and caching parameters for a JSON Web Key.
type JwkKey struct {
	// TTL configures how long a key version remains available after creation.
	//
	// A key signs for up to Rotation + Cache. The TTL should exceed that by the longest time a
	// recipient verifies a token it signed: the token's TTL plus leeway, or longer when a flow
	// verifies tokens past their expiry.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	// Rotation configures the interval at which new versions of the key are created.
	// It should be significantly lower than the TTL.
	Rotation time.Duration `json:"rotation" yaml:"rotation"`
	// Cache configures how long a key is cached in memory before being refetched from the database.
	// It should be significantly lower than the TTL.
	Cache time.Duration `json:"cache" yaml:"cache"`
	// StaleIfError configures how long past Cache the last fetched keys stay in service while refetching
	// fails, so an outage shorter than it leaves signing and verification running. Zero reports the
	// failure as soon as Cache expires.
	//
	// The newest cached key keeps signing through an outage. Rotation + Cache + StaleIfError should stay
	// below the TTL, which keeps that key unexpired once fetching recovers.
	StaleIfError time.Duration `json:"staleIfError" yaml:"staleIfError"`
	// StaleWhileRevalidate configures how long past Cache the cached keys keep serving while one
	// background fetch refreshes them, so no request waits on the database when the cache expires.
	// Zero makes the requests that find the cache expired wait for the refresh.
	StaleWhileRevalidate time.Duration `json:"staleWhileRevalidate" yaml:"staleWhileRevalidate"`
	// UnknownKeyIDInterval bounds how often a verifier refetches when it meets a key id its cache
	// does not hold, as a just-rotated key does before Cache expires. A caller sending unknown ids
	// gets one fetch per interval. Zero uses the jwt default.
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

// Target returns the issuer, audience and subject a usage stamps on its tokens. Signing and
// verification both read it, so a token verifies against the target it was signed with.
func (token JwkToken) Target() jwt.TargetConfig {
	return jwt.TargetConfig{
		Issuer:   token.Issuer,
		Audience: jwa.Audience{token.Audience},
		Subject:  token.Subject,
	}
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
