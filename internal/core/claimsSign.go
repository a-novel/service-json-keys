package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
)

// ClaimsSignRequest holds the parameters for a [ClaimsSign.Exec] call.
type ClaimsSignRequest struct {
	// Claims is the caller's payload, any JSON-serializable value. The service adds the standard JWT
	// claims before signing.
	Claims any
	// Usage identifies the key and token parameters to use for signing. See [jwkconfig.Usage].
	Usage string
}

// A ClaimsSign signs a set of claims and returns a compact JWT. The signing key and all token
// parameters are determined by the requested usage.
type ClaimsSign struct {
	usages map[string]claimsSignUsage
}

// claimsSignUsage holds what [ClaimsSign] signs the tokens of one usage with.
type claimsSignUsage struct {
	// claims sets the standard JWT claims. Verification checks them with the signature.
	claims   jwt.ClaimsProducerConfig
	producer *jwt.Producer
}

// NewClaimsSign creates a ClaimsSign service. Producers provide the per-usage signing plugins
// (see [NewJwkProducers]); keysConfig provides the token parameters for each usage. A usage signs
// only when both hold it.
func NewClaimsSign(
	producers map[string][]jwt.ProducerPlugin,
	keysConfig map[string]*jwkconfig.Usage,
) *ClaimsSign {
	usages := make(map[string]claimsSignUsage, len(keysConfig))

	for usage, keyConfig := range keysConfig {
		plugins, ok := producers[usage]
		if !ok {
			continue
		}

		usages[usage] = claimsSignUsage{
			claims:   jwt.ClaimsProducerConfig{TargetConfig: keyConfig.Token.Target(), TTL: keyConfig.Token.TTL},
			producer: jwt.NewProducer(jwt.ProducerConfig{Plugins: plugins}),
		}
	}

	return &ClaimsSign{usages: usages}
}

func (service *ClaimsSign) Exec(ctx context.Context, request *ClaimsSignRequest) (string, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ClaimsSign")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	signer, ok := service.usages[request.Usage]
	if !ok {
		return "", otel.ReportError(span, fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage))
	}

	claims, err := jwt.NewBasicClaims(request.Claims, signer.claims)
	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("create claims: %w", err))
	}

	// A caller claim that collides with the envelope fails inside Issue, at encoding time. The
	// wrapped error names the colliding members.
	token, err := signer.producer.Issue(ctx, claims, nil)
	if errors.Is(err, jwa.ErrReservedMember) {
		return "", otel.ReportError(span, fmt.Errorf("%w: %w", ErrReservedClaim, err))
	}

	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("issue token: %w", err))
	}

	return token, nil
}
