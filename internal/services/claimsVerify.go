package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwp"

	"github.com/a-novel/service-json-keys/internal/config"
)

type ClaimsVerifyRequest struct {
	Token         string
	Usage         string
	IgnoreExpired bool
}

type ClaimsVerify[Out any] struct {
	recipients map[string][]jwt.RecipientPlugin
	keys       map[string]*config.Jwk
}

func NewClaimsVerify[Out any](
	recipients map[string][]jwt.RecipientPlugin,
	keys map[string]*config.Jwk,
) *ClaimsVerify[Out] {
	return &ClaimsVerify[Out]{recipients: recipients, keys: keys}
}

func (service *ClaimsVerify[Out]) Exec(ctx context.Context, request *ClaimsVerifyRequest) (*Out, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ClaimsVerify")
	defer span.End()

	span.SetAttributes(attribute.String("usage", request.Usage))

	keyConfig, ok := service.keys[request.Usage]
	if !ok {
		return nil, otel.ReportError(span, ErrConfigNotFound)
	}

	var claims Out

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
