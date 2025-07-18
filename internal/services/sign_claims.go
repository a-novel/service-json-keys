package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt"

	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/config"
)

type SignClaimsRequest struct {
	// Claims to sign.
	Claims any
	// Usage of the claims, used to determine the signing key.
	Usage models.KeyUsage
}

type SignClaimsService struct {
	producers map[models.KeyUsage][]jwt.ProducerPlugin
	keys      map[models.KeyUsage]*config.JWKS
}

func NewSignClaimsService(
	producers map[models.KeyUsage][]jwt.ProducerPlugin,
	keys map[models.KeyUsage]*config.JWKS,
) *SignClaimsService {
	return &SignClaimsService{producers: producers, keys: keys}
}

func (service *SignClaimsService) SignClaims(ctx context.Context, request SignClaimsRequest) (string, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SignClaims")
	defer span.End()

	span.SetAttributes(attribute.String("usage", request.Usage.String()))

	keyConfig, ok := service.keys[request.Usage]
	if !ok {
		return "", otel.ReportError(span, fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
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
		return "", otel.ReportError(span, fmt.Errorf("create claims: %w", err))
	}

	producerPlugins, ok := service.producers[request.Usage]
	if !ok {
		return "", otel.ReportError(span, fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
	}

	producer := jwt.NewProducer(jwt.ProducerConfig{Plugins: producerPlugins})

	token, err := producer.Issue(ctx, claims, nil)
	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("issue token: %w", err))
	}

	return otel.ReportSuccess(span, token), nil
}
