package servicejsonkeys

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/internal/core"
)

// KeyUsage identifies the intended purpose of a token. It selects the signing key and full
// token configuration the service uses when signing, and the corresponding public keys and
// parameters used to verify. Each producer service owns one or more usages, registered in
// the service configuration.
type KeyUsage = string

const (
	// KeyUsageAuth is used to produce access tokens for the authentication service.
	KeyUsageAuth KeyUsage = "auth"
	// KeyUsageAuthRefresh is used to produce refresh tokens for the authentication service.
	KeyUsageAuthRefresh KeyUsage = "auth-refresh"
)

// VerifyClaimsOptions configures optional behavior for a [ClaimsVerifier.VerifyClaims] call.
type VerifyClaimsOptions struct {
	// IgnoreExpired, when true, allows expired tokens to pass verification.
	IgnoreExpired bool
}

// VerifyClaimsRequest holds the parameters for a [ClaimsVerifier.VerifyClaims] call.
type VerifyClaimsRequest struct {
	// Usage is the key usage the token was signed for; must match the value used at signing time. See [KeyUsage].
	Usage KeyUsage
	// AccessToken is the compact JWT to verify: the dot-separated base64url string
	// (header.payload.signature) returned by the ClaimsSign RPC.
	AccessToken string
	// Options configures optional verification behavior for this call; nil means all defaults apply.
	Options *VerifyClaimsOptions
}

// A ClaimsVerifier verifies a compact JWT and deserializes its payload into C.
// Verification is performed locally using public keys sourced from the [Client]; no network
// call is made per verification. Obtain one with [NewClaimsVerifier].
type ClaimsVerifier[C any] interface {
	// VerifyClaims verifies the compact JWT in req and, if valid, returns the decoded claims.
	VerifyClaims(ctx context.Context, req *VerifyClaimsRequest) (*C, error)
}

type claimsVerifier[C any] struct {
	service *core.ClaimsVerify[C]
}

// NewClaimsVerifier creates a token verifier backed by the key configuration carried by c. It
// builds the cached public-key sources used for local verification, returning an error if the
// configuration references an unsupported algorithm. C must be JSON-serializable and match the
// claims type used when signing for the same usage.
func NewClaimsVerifier[C any](c Client) (ClaimsVerifier[C], error) {
	// Build the cached public-key sources here rather than in the client, so the client's exported
	// surface never mentions a jwt type. A sign-only consumer that never calls this pays nothing.
	adapter := newJwkExportGrpc(c)

	sources, err := core.NewJwkPublicSource(adapter, c.Keys())
	if err != nil {
		return nil, fmt.Errorf("(NewClaimsVerifier) new public sources: %w", err)
	}

	recipients, err := core.NewJwkRecipients(sources, c.Keys())
	if err != nil {
		return nil, fmt.Errorf("(NewClaimsVerifier) new recipients: %w", err)
	}

	return &claimsVerifier[C]{service: core.NewClaimsVerify[C](recipients, c.Keys())}, nil
}

func (verifier *claimsVerifier[C]) VerifyClaims(ctx context.Context, req *VerifyClaimsRequest) (*C, error) {
	return verifier.service.Exec(ctx, &core.ClaimsVerifyRequest{
		Token:         req.AccessToken,
		Usage:         req.Usage,
		IgnoreExpired: lo.FromPtr(req.Options).IgnoreExpired,
	})
}
