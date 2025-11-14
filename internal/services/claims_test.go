package services_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func TestSignAndVerifyClaims(t *testing.T) {
	t.Parallel()

	privateKeys, publicKeys := generateAuthTokenKeySet(t, 1)

	privateKeysJSON := lo.Map(privateKeys, func(item *jwk.Key[ed25519.PrivateKey], _ int) *jwa.JWK {
		return item.JWK
	})

	publicKeysJSON := lo.Map(publicKeys, func(item *jwk.Key[ed25519.PublicKey], _ int) *jwa.JWK {
		return item.JWK
	})

	type testClaims struct {
		Foo string `json:"foo"`
	}

	testConfig := map[string]*config.Jwk{
		"test-usage": {
			Alg: jwa.EdDSA,
			Key: config.JwkKey{
				TTL:      168 * time.Hour,
				Rotation: 24 * time.Hour,
				Cache:    30 * time.Minute,
			},
			Token: config.JwkToken{
				TTL:      24 * time.Hour,
				Issuer:   "test-issuer",
				Audience: "test-audience",
				Subject:  "test-subject",
				Leeway:   5 * time.Minute,
			},
		},
	}

	producers, err := services.NewJwkProducers(&services.JwkPrivateSources{
		EdDSA: map[string]*jwk.Source[ed25519.PrivateKey]{
			"test-usage": jwk.NewED25519PrivateSource(jwk.SourceConfig{
				Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
					return privateKeysJSON, nil
				},
			}),
		},
		HMAC: make(map[string]*jwk.Source[[]byte]),
		ES:   make(map[string]*jwk.Source[*ecdsa.PrivateKey]),
		RSA:  make(map[string]*jwk.Source[*rsa.PrivateKey]),
	}, testConfig)
	require.NoError(t, err)

	recipients, err := services.NewJwkRecipients(&services.JwkPublicSources{
		EdDSA: map[string]*jwk.Source[ed25519.PublicKey]{
			"test-usage": jwk.NewED25519PublicSource(jwk.SourceConfig{
				Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
					return publicKeysJSON, nil
				},
			}),
		},
		HMAC: make(map[string]*jwk.Source[[]byte]),
		ES:   make(map[string]*jwk.Source[*ecdsa.PublicKey]),
		RSA:  make(map[string]*jwk.Source[*rsa.PublicKey]),
	}, testConfig)
	require.NoError(t, err)

	testCases := []struct {
		name string

		claims *testClaims
	}{
		{
			name: "SimpleClaims",

			claims: &testClaims{Foo: "bar"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			signer := services.NewClaimsSign(producers, testConfig)
			verifier := services.NewClaimsVerify[testClaims](recipients, testConfig)

			ctx := context.Background()

			signedClaims, err := signer.Exec(ctx, &services.ClaimsSignRequest{
				Claims: testCase.claims,
				Usage:  "test-usage",
			})
			require.NoError(t, err)

			verifiedClaims, err := verifier.Exec(ctx, &services.ClaimsVerifyRequest{
				Token: signedClaims,
				Usage: "test-usage",
			})
			require.NoError(t, err)

			require.Equal(t, testCase.claims, verifiedClaims)
		})
	}
}
