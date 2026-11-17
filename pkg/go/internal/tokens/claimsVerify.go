package tokens

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwp"

	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
)

// ClaimsVerifyRequest holds the parameters for a [ClaimsVerify.Exec] call.
type ClaimsVerifyRequest struct {
	// Token is the compact JWT to verify.
	Token string
	// Usage is the key usage the token was signed under; must match the value used at signing time.
	Usage string
	// IgnoreExpired allows expired tokens to pass verification. Useful for refresh flows.
	IgnoreExpired bool
}

// A ClaimsVerify verifies a signed JWT and decodes its claims into Out, validating
// all token claims against the configuration registered for the given usage.
type ClaimsVerify[Out any] struct {
	usages map[string]claimsVerifyUsage
}

// claimsVerifyUsage holds the recipients [ClaimsVerify] verifies the tokens of one usage with.
// Both check the usage's target, and recipient also checks the token's expiry.
type claimsVerifyUsage struct {
	recipient        *jwt.Recipient
	expiredRecipient *jwt.Recipient
}

// NewClaimsVerify creates a ClaimsVerify service. Recipients provide the per-usage verification
// plugins (see [NewJwkRecipients]); keysConfig provides the token parameters for each usage. A
// usage verifies only when both hold it.
func NewClaimsVerify[Out any](
	recipients map[string][]jwt.RecipientPlugin,
	keysConfig map[string]*jwkconfig.Usage,
) *ClaimsVerify[Out] {
	usages := make(map[string]claimsVerifyUsage, len(keysConfig))

	for usage, keyConfig := range keysConfig {
		plugins, ok := recipients[usage]
		if !ok {
			continue
		}

		target := jwp.NewClaimsCheckTarget(keyConfig.Token.Target())
		expiry := jwp.NewClaimsCheckTimestamp(keyConfig.Token.Leeway, true)

		usages[usage] = claimsVerifyUsage{
			recipient:        newClaimsRecipient(plugins, target, expiry),
			expiredRecipient: newClaimsRecipient(plugins, target),
		}
	}

	return &ClaimsVerify[Out]{usages: usages}
}

// Exec authenticates the token and decodes its claims using the configured usage policy.
func (service *ClaimsVerify[Out]) Exec(ctx context.Context, request *ClaimsVerifyRequest) (*Out, error) {
	ctx, span := otel.Tracer().Start(ctx, "tokens.ClaimsVerify")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	verifier, ok := service.usages[request.Usage]
	if !ok {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", jwkconfig.ErrConfigNotFound, request.Usage))
	}

	recipient := lo.Ternary(request.IgnoreExpired, verifier.expiredRecipient, verifier.recipient)

	var claims Out

	err := recipient.Consume(ctx, request.Token, &claims)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return &claims, nil
}

// newClaimsRecipient returns a recipient that verifies a token with plugins, then runs checks on
// its claims before decoding them.
func newClaimsRecipient(plugins []jwt.RecipientPlugin, checks ...jwp.ClaimsCheck) *jwt.Recipient {
	checker := jwp.NewClaimsChecker(&jwp.ClaimsCheckerConfig{Checks: checks})

	return jwt.NewRecipient(jwt.RecipientConfig{Plugins: plugins, Deserializer: checker.Unmarshal})
}
