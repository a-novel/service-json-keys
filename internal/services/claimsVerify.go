package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwp"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

type ClaimsVerifyRequest struct {
	// The full token to verify.
	Token string
	// The intended usage of the token. It must match the usage that has been used for its
	// creation.
	Usage string
	// Tell the service to ignore expiry date verification. This can be used to validate
	// expired tokens and perform operations like token refresh.
	IgnoreExpired bool
}

// ClaimsVerify is a service used to verify and decode a token. It returns the decoded claims.
type ClaimsVerify[Out any] struct {
	recipients map[string][]jwt.RecipientPlugin
	keysConfig map[string]*config.Jwk
}

// NewClaimsVerify creates a new ClaimsVerify service.
//
// The recipients are a list of plugins to use depending on the key usage.
//
// This method also requires to provide JSON key configuration for each usage.
func NewClaimsVerify[Out any](
	recipients map[string][]jwt.RecipientPlugin,
	keysConfig map[string]*config.Jwk,
) *ClaimsVerify[Out] {
	return &ClaimsVerify[Out]{recipients: recipients, keysConfig: keysConfig}
}

func (service *ClaimsVerify[Out]) Exec(ctx context.Context, request *ClaimsVerifyRequest) (*Out, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ClaimsVerify")
	defer span.End()

	span.SetAttributes(attribute.String("usage", request.Usage))

	keyConfig, ok := service.keysConfig[request.Usage]
	if !ok {
		return nil, otel.ReportError(span, ErrConfigNotFound)
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
