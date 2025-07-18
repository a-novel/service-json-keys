package services_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/config"
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

			producers, err := services.NewProducers(&services.PrivateKeysSourceType{
				EdDSA: map[models.KeyUsage]*jwk.Source[ed25519.PrivateKey]{
					models.KeyUsageAuth: jwk.NewED25519PrivateSource(jwk.SourceConfig{
						Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
							return privateKeysJSON, nil
						},
					}),
				},
				HMAC: make(map[models.KeyUsage]*jwk.Source[[]byte]),
				ES:   make(map[models.KeyUsage]*jwk.Source[*ecdsa.PrivateKey]),
				RSA:  make(map[models.KeyUsage]*jwk.Source[*rsa.PrivateKey]),
			}, config.JWKSPresetDefault)
			require.NoError(t, err)

			recipients, err := services.NewRecipients(&services.PublicKeySourceType{
				EdDSA: map[models.KeyUsage]*jwk.Source[ed25519.PublicKey]{
					models.KeyUsageAuth: jwk.NewED25519PublicSource(jwk.SourceConfig{
						Fetch: func(_ context.Context) ([]*jwa.JWK, error) {
							return publicKeysJSON, nil
						},
					}),
				},
				HMAC: make(map[models.KeyUsage]*jwk.Source[[]byte]),
				ES:   make(map[models.KeyUsage]*jwk.Source[*ecdsa.PublicKey]),
				RSA:  make(map[models.KeyUsage]*jwk.Source[*rsa.PublicKey]),
			}, config.JWKSPresetDefault)
			require.NoError(t, err)

			signer := services.NewSignClaimsService(producers, config.JWKSPresetDefault)
			verifier := services.NewVerifyClaimsService[testClaims](recipients, config.JWKSPresetDefault)

			ctx := context.Background()

			signedClaims, err := signer.SignClaims(ctx, services.SignClaimsRequest{
				Claims: testCase.claims,
				Usage:  models.KeyUsageAuth,
			})
			require.NoError(t, err)

			verifiedClaims, err := verifier.VerifyClaims(ctx, services.VerifyClaimsRequest{
				Token: signedClaims,
				Usage: models.KeyUsageAuth,
			})
			require.NoError(t, err)

			require.Equal(t, testCase.claims, verifiedClaims)
		})
	}
}
