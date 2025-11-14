package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

type ClaimsSignRequest struct {
	Claims any
	Usage  string
}

type ClaimsSign struct {
	producers map[string][]jwt.ProducerPlugin
	keys      map[string]*config.Jwk
}

func NewClaimsSign(
	producers map[string][]jwt.ProducerPlugin,
	keys map[string]*config.Jwk,
) *ClaimsSign {
	return &ClaimsSign{producers: producers, keys: keys}
}

func (service *ClaimsSign) Exec(ctx context.Context, request *ClaimsSignRequest) (string, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ClaimsSign")
	defer span.End()

	span.SetAttributes(attribute.String("usage", request.Usage))

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
