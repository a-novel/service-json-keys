package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

// ClaimsSignRequest holds the parameters for a [ClaimsSign.Exec] call.
type ClaimsSignRequest struct {
	// Claims is the caller-supplied payload to embed in the JWT. Any JSON-serializable value
	// is accepted; the service adds the standard JWT claim envelope before signing.
	Claims any
	// Usage identifies the key and token parameters to use for signing. See [config.Jwk].
	Usage string
}

// A ClaimsSign signs a set of claims and returns a compact JWT. The signing key and all token
// parameters are determined by the requested usage.
type ClaimsSign struct {
	producers  map[string][]jwt.ProducerPlugin
	keysConfig map[string]*config.Jwk
}

// NewClaimsSign creates a ClaimsSign service. Producers provide the per-usage signing plugins
// (see [NewJwkProducers]); keysConfig provides the token parameters for each usage.
func NewClaimsSign(
	producers map[string][]jwt.ProducerPlugin,
	keysConfig map[string]*config.Jwk,
) *ClaimsSign {
	return &ClaimsSign{producers: producers, keysConfig: keysConfig}
}

func (service *ClaimsSign) Exec(ctx context.Context, request *ClaimsSignRequest) (string, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ClaimsSign")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	keyConfig, ok := service.keysConfig[request.Usage]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage)
	}

	// Attach the standard JWT claim envelope from the usage config; these claims are
	// checked during verification alongside the token signature.
	claims, err := jwt.NewBasicClaims(request.Claims, jwt.ClaimsProducerConfig{
		TargetConfig: jwt.TargetConfig{
			Issuer:   keyConfig.Token.Issuer,
			Audience: jwa.Audience{keyConfig.Token.Audience},
			Subject:  keyConfig.Token.Subject,
		},
		TTL: keyConfig.Token.TTL,
	})
	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("create claims: %w", err))
	}

	producerPlugins, ok := service.producers[request.Usage]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage)
	}

	producer := jwt.NewProducer(jwt.ProducerConfig{Plugins: producerPlugins})

	// A caller claim that collides with the envelope above is rejected at
	// encoding time, inside Issue. The request is malformed, so it is classified
	// rather than reported as a fault; the wrapped error names the members.
	token, err := producer.Issue(ctx, claims, nil)
	if errors.Is(err, jwa.ErrReservedMember) {
		return "", fmt.Errorf("%w: %w", ErrReservedClaim, err)
	}

	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("issue token: %w", err))
	}

	return otel.ReportSuccess(span, token), nil
}
