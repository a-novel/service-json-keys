package core_test

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
)

func TestClaimsSignAndVerify(t *testing.T) {
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

	testCases := []struct {
		name string

		claims *testClaims
	}{
		{
			name: "Success",

			claims: &testClaims{Foo: "bar"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			privateSource := coremocks.NewMockJwkPrivateSource(t)
			privateSource.EXPECT().
				SearchKeys(mock.Anything, "test-usage").
				Return(privateKeysJSON, nil).
				Once()

			producers, err := core.NewJwkProducers(privateSource, testConfig)
			require.NoError(t, err)

			publicSource := coremocks.NewMockJwkPublicSource(t)
			publicSource.EXPECT().
				SearchKeys(mock.Anything, "test-usage").
				Return(publicKeysJSON, nil).
				Once()

			recipients, err := core.NewJwkRecipients(publicSource, testConfig)
			require.NoError(t, err)

			signer := core.NewClaimsSign(producers, testConfig)
			verifier := core.NewClaimsVerify[testClaims](recipients, testConfig)

			signedClaims, err := signer.Exec(t.Context(), &core.ClaimsSignRequest{
				Claims: testCase.claims,
				Usage:  "test-usage",
			})
			require.NoError(t, err)

			verifiedClaims, err := verifier.Exec(t.Context(), &core.ClaimsVerifyRequest{
				Token: signedClaims,
				Usage: "test-usage",
			})
			require.NoError(t, err)

			require.Equal(t, testCase.claims, verifiedClaims)

			privateSource.AssertExpectations(t)
			publicSource.AssertExpectations(t)
		})
	}
}
