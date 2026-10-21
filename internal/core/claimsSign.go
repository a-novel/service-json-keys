package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
)

// ClaimsSignRequest holds the parameters for a [ClaimsSign.Exec] call.
type ClaimsSignRequest struct {
	// Claims is the caller's payload, any JSON-serializable value. The service adds the standard JWT
	// claims before signing.
	Claims any
	// Usage identifies the key and token parameters to use for signing. See [jwkconfig.Jwk].
	Usage string
}

// A ClaimsSign signs a set of claims and returns a compact JWT. The signing key and all token
// parameters are determined by the requested usage.
type ClaimsSign struct {
	producers  map[string][]jwt.ProducerPlugin
	keysConfig map[string]*jwkconfig.Jwk
}

// NewClaimsSign creates a ClaimsSign service. Producers provide the per-usage signing plugins
// (see [NewJwkProducers]); keysConfig provides the token parameters for each usage.
func NewClaimsSign(
	producers map[string][]jwt.ProducerPlugin,
	keysConfig map[string]*jwkconfig.Jwk,
) *ClaimsSign {
	return &ClaimsSign{producers: producers, keysConfig: keysConfig}
}

func (service *ClaimsSign) Exec(ctx context.Context, request *ClaimsSignRequest) (string, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ClaimsSign")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	keyConfig, ok := service.keysConfig[request.Usage]
	if !ok {
		return "", otel.ReportError(span, fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
	}

	// The usage config sets the standard JWT claims. Verification checks them with the signature.
	claimsConfig := jwt.ClaimsProducerConfig{
		TargetConfig: tokenTarget(keyConfig.Token),
		TTL:          keyConfig.Token.TTL,
	}

	claims, err := jwt.NewBasicClaims(request.Claims, claimsConfig)
	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("create claims: %w", err))
	}

	producerPlugins, ok := service.producers[request.Usage]
	if !ok {
		return "", otel.ReportError(span, fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
	}

	producer := jwt.NewProducer(jwt.ProducerConfig{Plugins: producerPlugins})

	// A caller claim that collides with the envelope fails inside Issue, at encoding time. The
	// wrapped error names the colliding members.
	token, err := producer.Issue(ctx, claims, nil)
	if errors.Is(err, jwa.ErrReservedMember) {
		return "", otel.ReportError(span, fmt.Errorf("%w: %w", ErrReservedClaim, err))
	}

	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("issue token: %w", err))
	}

	return token, nil
}
