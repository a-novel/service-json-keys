package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwp"

	"github.com/a-novel/service-json-keys/v2/internal/config"
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
	keysConfig map[string]*config.Jwk
}

// NewClaimsVerify creates a ClaimsVerify service. Recipients provide the per-usage verification
// plugins (see [NewJwkRecipients]); keysConfig provides the token parameters for each usage.
func NewClaimsVerify[Out any](
	recipients map[string][]jwt.RecipientPlugin,
	keysConfig map[string]*config.Jwk,
) *ClaimsVerify[Out] {
	return &ClaimsVerify[Out]{recipients: recipients, keysConfig: keysConfig}
}

func (service *ClaimsVerify[Out]) Exec(ctx context.Context, request *ClaimsVerifyRequest) (*Out, error) {
	ctx, span := otel.Tracer().Start(ctx, "services.ClaimsVerify")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	keyConfig, ok := service.keysConfig[request.Usage]
	if !ok {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
	}

	var claims Out

	// Use the key configuration to further validate the token claims.
	checks := []jwp.ClaimsCheck{
		jwp.NewClaimsCheckTarget(jwt.TargetConfig{
			Issuer:   keyConfig.Token.Issuer,
			Audience: keyConfig.Token.Audience,
			Subject:  keyConfig.Token.Subject,
		}),
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
			fmt.Errorf("%w: no recipients found for usage %s", ErrConfigNotFound, request.Usage),
		)
	}

	recipient := jwt.NewRecipient(
		jwt.RecipientConfig{
			Plugins:      recipientPlugins,
			Deserializer: deserializer.Unmarshal,
		})

	err := recipient.Consume(ctx, request.Token, &claims)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &claims), nil
}
