package tokens

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwp"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
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
	recipients map[string][]jwt.RecipientPlugin
	keysConfig map[string]*jwkconfig.Jwk
}

// NewClaimsVerify creates a ClaimsVerify service. Recipients provide the per-usage verification
// plugins (see [NewJwkRecipients]); keysConfig provides the token parameters for each usage.
func NewClaimsVerify[Out any](
	recipients map[string][]jwt.RecipientPlugin,
	keysConfig map[string]*jwkconfig.Jwk,
) *ClaimsVerify[Out] {
	return &ClaimsVerify[Out]{recipients: recipients, keysConfig: keysConfig}
}

// Exec authenticates the token and decodes its claims using the configured usage policy.
func (service *ClaimsVerify[Out]) Exec(ctx context.Context, request *ClaimsVerifyRequest) (*Out, error) {
	ctx, span := otel.Tracer().Start(ctx, "tokens.ClaimsVerify")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	keyConfig, ok := service.keysConfig[request.Usage]
	if !ok {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", jwkconfig.ErrConfigNotFound, request.Usage))
	}

	// Validate the claims against the usage's configured target, and its expiry unless waived.
	checks := []jwp.ClaimsCheck{
		jwp.NewClaimsCheckTarget(keyConfig.Token.Target()),
	}
	if !request.IgnoreExpired {
		checks = append(checks, jwp.NewClaimsCheckTimestamp(keyConfig.Token.Leeway, true))
	}

	deserializer := jwp.NewClaimsChecker(&jwp.ClaimsCheckerConfig{
		Checks: checks,
	})

	recipientPlugins, ok := service.recipients[request.Usage]
	if !ok {
		return nil, otel.ReportError(span,
			fmt.Errorf("%w: no recipients found for usage %s", jwkconfig.ErrConfigNotFound, request.Usage),
		)
	}

	recipientConfig := jwt.RecipientConfig{
		Plugins:      recipientPlugins,
		Deserializer: deserializer.Unmarshal,
	}

	recipient := jwt.NewRecipient(recipientConfig)

	var claims Out

	err := recipient.Consume(ctx, request.Token, &claims)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return &claims, nil
}
