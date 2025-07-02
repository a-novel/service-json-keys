package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel-kit/jwt"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/models"
)

var ErrSignClaimsService = errors.New("SignClaimsService.SignClaims")

func NewErrSignClaimsService(err error) error {
	return errors.Join(err, ErrSignClaimsService)
}

type SignClaimsRequest struct {
	// Claims to sign.
	Claims any
	// Usage of the claims, used to determine the signing key.
	Usage models.KeyUsage
}

type SignClaimsService struct {
	producers map[models.KeyUsage][]jwt.ProducerPlugin
}

func NewSignClaimsService(producers map[models.KeyUsage][]jwt.ProducerPlugin) *SignClaimsService {
	return &SignClaimsService{producers: producers}
}

func (service *SignClaimsService) SignClaims(ctx context.Context, request SignClaimsRequest) (string, error) {
	span := sentry.StartSpan(ctx, "SignClaimsService.SignClaims")
	defer span.Finish()

	span.SetData("usage", request.Usage)

	keyConfig, ok := config.Keys[request.Usage]
	if !ok {
		return "", NewErrSignClaimsService(fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
	}

	claims, err := jwt.NewBasicClaims(request.Claims, jwt.ClaimsProducerConfig{
		TargetConfig: jwt.TargetConfig{
			Issuer:   keyConfig.Token.Issuer,
			Audience: keyConfig.Token.Audience,
			Subject:  keyConfig.Token.Subject,
		},
		TTL: keyConfig.Token.TTL,
	})
	if err != nil {
		span.SetData("error", err.Error())

		return "", NewErrSignClaimsService(fmt.Errorf("create claims: %w", err))
	}

	producerPlugins, ok := service.producers[request.Usage]
	if !ok {
		span.SetData("error", fmt.Sprintf("no producer plugins found for usage: %s", request.Usage))

		return "", NewErrSignClaimsService(fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
	}

	producer := jwt.NewProducer(jwt.ProducerConfig{Plugins: producerPlugins})

	token, err := producer.Issue(span.Context(), claims, nil)
	if err != nil {
		span.SetData("error", err.Error())

		return "", NewErrSignClaimsService(fmt.Errorf("issue token: %w", err))
	}

	return token, nil
}
