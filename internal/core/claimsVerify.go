package core

import (
	"github.com/a-novel-kit/jwt/v2"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core/verifier"
)

// ClaimsVerifyRequest holds the parameters for a [ClaimsVerify.Exec] call.
type ClaimsVerifyRequest = verifier.ClaimsVerifyRequest

// ClaimsVerify verifies signed JWTs using the shared core verification service.
type ClaimsVerify[Out any] = verifier.ClaimsVerify[Out]

// NewClaimsVerify creates a verifier using per-usage plugins and token configuration.
func NewClaimsVerify[Out any](
	recipients map[string][]jwt.RecipientPlugin,
	keysConfig map[string]*config.Jwk,
) *ClaimsVerify[Out] {
	return verifier.NewClaimsVerify[Out](recipients, keysConfig)
}
