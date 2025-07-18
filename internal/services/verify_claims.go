package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwp"

	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/config"
)

type VerifyClaimsRequest struct {
	Token         string
	Usage         models.KeyUsage
	IgnoreExpired bool
}

type VerifyClaimsService[Out any] struct {
	recipients map[models.KeyUsage][]jwt.RecipientPlugin
	keys       map[models.KeyUsage]*config.JWKS
}

func NewVerifyClaimsService[Out any](
	recipients map[models.KeyUsage][]jwt.RecipientPlugin,
	keys map[models.KeyUsage]*config.JWKS,
) *VerifyClaimsService[Out] {
	return &VerifyClaimsService[Out]{recipients: recipients, keys: keys}
}

func (service *VerifyClaimsService[Out]) VerifyClaims(ctx context.Context, request VerifyClaimsRequest) (*Out, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.VerifyClaims")
	defer span.End()

	span.SetAttributes(attribute.String("usage", request.Usage.String()))

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
