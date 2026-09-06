package verifier

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwp"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/jwk"
)

// ErrConfigNotFound is returned when no verification configuration exists for a usage.
var ErrConfigNotFound = errors.New("no config found for the requested usage")

// Request holds the token and verification settings for a call to [Claims.Verify].
type Request struct {
	// Token is the compact JWT to verify.
	Token string
	// Usage selects the key and registered-claim configuration used to verify the token.
	Usage string
	// IgnoreExpired allows an expired token to pass timestamp validation.
	IgnoreExpired bool
}

// Claims verifies signed JWTs and decodes their application claims into Out.
type Claims[Out any] struct {
	recipients Recipients
	keys       map[string]*jwkconfig.Jwk
}

// NewClaims creates a local claims verifier from key recipients and usage configuration.
func NewClaims[Out any](recipients Recipients, keys map[string]*jwkconfig.Jwk) *Claims[Out] {
	return &Claims[Out]{recipients: recipients, keys: keys}
}

// Verify authenticates request.Token and decodes its claims.
func (claimsVerifier *Claims[Out]) Verify(ctx context.Context, request *Request) (*Out, error) {
	ctx, span := otel.Tracer().Start(ctx, "verifier.Claims")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	keyConfig, ok := claimsVerifier.keys[request.Usage]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, request.Usage)
	}

	checks := []jwp.ClaimsCheck{
		jwp.NewClaimsCheckTarget(jwt.TargetConfig{
			Issuer:   keyConfig.Token.Issuer,
			Audience: jwa.Audience{keyConfig.Token.Audience},
			Subject:  keyConfig.Token.Subject,
		}),
	}
	if !request.IgnoreExpired {
		checks = append(checks, jwp.NewClaimsCheckTimestamp(keyConfig.Token.Leeway, true))
	}

	var output Out

	recipientPlugins, ok := claimsVerifier.recipients[request.Usage]
	if !ok {
		return nil, fmt.Errorf("%w: no recipients found for usage %s", ErrConfigNotFound, request.Usage)
	}

	recipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins: recipientPlugins,
		Deserializer: jwp.NewClaimsChecker(&jwp.ClaimsCheckerConfig{
			Checks: checks,
		}).Unmarshal,
	})

	err := recipient.Consume(ctx, request.Token, &output)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &output), nil
}
