package pkg

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// KeyUsage determines the intended usage of a token. The token parameters (ttl, issuer, etc.),
// along with the set of keys used to generate it, are determined by the usage.
//
// Note that each service willing to produce tokens must register its own unique usages,
// and configure them directly on this service.
type KeyUsage = string

const (
	// KeyUsageAuth is used to produce access tokens for the authentication service.
	KeyUsageAuth KeyUsage = "auth"
	// KeyUsageAuthRefresh is used to produce refresh tokens for the authentication service.
	KeyUsageAuthRefresh KeyUsage = "auth-refresh"
)

type VerifyClaimsOptions struct {
	// Ignore expiration date check. This can be used to validate expired tokens.
	IgnoreExpired bool
}

type VerifyClaimsRequest struct {
	// See KeyUsage.
	Usage KeyUsage
	// The token to verify.
	AccessToken string
	// Validation options. Check VerifyClaimsOptions for more information.
	Options *VerifyClaimsOptions
}

// ClaimsVerifier is a service used to verify and decode a token. It returns the decoded claims.
type ClaimsVerifier[C any] interface {
	VerifyClaims(ctx context.Context, req *VerifyClaimsRequest) (*C, error)
}

type claimsVerifier[C any] struct {
	service *services.ClaimsVerify[C]
}

func NewClaimsVerifier[C any](c Client) ClaimsVerifier[C] {
	service := services.NewClaimsVerify[C](c.Recipients(), c.Keys())

	return &claimsVerifier[C]{service: service}
}

func (verifier *claimsVerifier[C]) VerifyClaims(ctx context.Context, req *VerifyClaimsRequest) (*C, error) {
	return verifier.service.Exec(ctx, &services.ClaimsVerifyRequest{
		Token:         req.AccessToken,
		Usage:         req.Usage,
		IgnoreExpired: lo.FromPtr(req.Options).IgnoreExpired,
	})
}
