package core

import (
	"errors"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

// ErrConfigNotFound is returned when no JWK configuration is registered for a given key usage.
var ErrConfigNotFound = errors.New("no config found for the requested usage")

// ErrJwkNotFound is returned when no active key matches the requested ID.
var ErrJwkNotFound = dao.ErrJwkSelectNotFound

// ErrReservedClaim is returned when the caller's claims name a registered JWT
// parameter. Those belong to the envelope the service stamps from the usage
// config, so a caller setting one is asking for a token that contradicts the
// terms it was signed under. It reports a malformed request, not a fault.
var ErrReservedClaim = errors.New("claims may not set a registered JWT parameter")

// tokenTarget returns the issuer, audience and subject a usage stamps on its tokens. Signing and
// verification both read it, so a token verifies against the target it was signed with.
func tokenTarget(token jwkconfig.JwkToken) jwt.TargetConfig {
	return jwt.TargetConfig{
		Issuer:   token.Issuer,
		Audience: jwa.Audience{token.Audience},
		Subject:  token.Subject,
	}
}
