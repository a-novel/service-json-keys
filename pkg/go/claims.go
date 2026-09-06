package servicejsonkeys

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/internal/verifier"
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
	claims *verifier.Claims[C]
}

// NewClaimsVerifier creates a token verifier backed by the key configuration carried by c. It
// builds the cached public-key sources used for local verification, returning an error if the
// configuration references an unsupported algorithm. C must be JSON-serializable and match the
// claims type used when signing for the same usage.
func NewClaimsVerifier[C any](c Client) (ClaimsVerifier[C], error) {
	// Building the public-key sources here keeps jwt types off the client's exported surface,
	// and a sign-only consumer never pays for the setup.
	adapter := newJwkExportGrpc(c)

	recipients, err := verifier.NewRecipients(adapter, c.Keys())
	if err != nil {
		return nil, fmt.Errorf("(NewClaimsVerifier) new recipients: %w", err)
	}

	return &claimsVerifier[C]{claims: verifier.NewClaims[C](recipients, c.Keys())}, nil
}

func (claimsVerifier *claimsVerifier[C]) VerifyClaims(ctx context.Context, req *VerifyClaimsRequest) (*C, error) {
	return claimsVerifier.claims.Verify(ctx, &verifier.Request{
		Token:         req.AccessToken,
		Usage:         req.Usage,
		IgnoreExpired: lo.FromPtr(req.Options).IgnoreExpired,
	})
}
