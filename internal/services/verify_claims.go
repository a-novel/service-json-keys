package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwp"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/models"
)

var ErrVerifyClaimsService = errors.New("VerifyClaimsService.VerifyClaims")

func NewErrVerifyClaimsService(err error) error {
	return errors.Join(err, ErrVerifyClaimsService)
}

type VerifyClaimsRequest struct {
	Token         string
	Usage         models.KeyUsage
	IgnoreExpired bool
}

type VerifyClaimsService[Out any] struct {
	recipients map[models.KeyUsage][]jwt.RecipientPlugin
}

func NewVerifyClaimsService[Out any](recipients map[models.KeyUsage][]jwt.RecipientPlugin) *VerifyClaimsService[Out] {
	return &VerifyClaimsService[Out]{recipients: recipients}
}

func (service *VerifyClaimsService[Out]) VerifyClaims(ctx context.Context, request VerifyClaimsRequest) (*Out, error) {
	span := sentry.StartSpan(ctx, "VerifyClaimsService.VerifyClaims")
	defer span.Finish()

	span.SetData("usage", request.Usage)

	keyConfig, ok := config.Keys[request.Usage]
	if !ok {
		return nil, NewErrVerifyClaimsService(ErrConfigNotFound)
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
		return nil, NewErrVerifyClaimsService(
			fmt.Errorf("%w: no recipients found for usage %s", ErrConfigNotFound, request.Usage),
		)
	}

	recipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins:      recipientPlugins,
		Deserializer: deserializer.Unmarshal,
	})

	err := recipient.Consume(span.Context(), request.Token, &claims)
	if err != nil {
		span.SetData("error", err.Error())

		return nil, NewErrVerifyClaimsService(err)
	}

	return &claims, nil
}
