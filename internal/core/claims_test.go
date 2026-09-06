package core_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
)

func loadGeneratedJWK(t *testing.T, key any) *jwa.JWK {
	t.Helper()

	encoded, err := json.Marshal(key)
	require.NoError(t, err)

	var output jwa.JWK
	require.NoError(t, json.Unmarshal(encoded, &output))

	return &output
}

func TestClaimsSignAndVerify(t *testing.T) {
	t.Parallel()

	type testClaims struct {
		Foo string `json:"foo"`
	}

	testCases := []struct {
		name string
		alg  jwa.Alg
	}{
		{name: "Success/EdDSA", alg: jwa.EdDSA},
		{name: "Success/ES256", alg: jwa.ES256},
		{name: "Success/ES384", alg: jwa.ES384},
		{name: "Success/ES512", alg: jwa.ES512},
		{name: "Success/RS256", alg: jwa.RS256},
		{name: "Success/RS384", alg: jwa.RS384},
		{name: "Success/RS512", alg: jwa.RS512},
		{name: "Success/PS256", alg: jwa.PS256},
		{name: "Success/PS384", alg: jwa.PS384},
		{name: "Success/PS512", alg: jwa.PS512},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			generator, ok := core.JwkGenerators[testCase.alg]
			require.True(t, ok)

			generatedKey, err := generator()
			require.NoError(t, err)

			privateKey := loadGeneratedJWK(t, generatedKey.PrivateKey)
			publicKey := loadGeneratedJWK(t, generatedKey.PublicKey)
			require.Equal(t, generatedKey.PrivateKID, privateKey.KID)
			require.Equal(t, generatedKey.PublicKID, publicKey.KID)

			keys := map[string]*config.Jwk{
				"test-usage": {
					Alg: testCase.alg,
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

			privateSource := coremocks.NewMockJwkPrivateSource(t)
			privateSource.EXPECT().
				SearchKeys(mock.Anything, "test-usage").
				Return([]*jwa.JWK{privateKey}, nil).
				Once()

			producers, err := core.NewJwkProducers(privateSource, keys)
			require.NoError(t, err)

			publicSource := coremocks.NewMockJwkPublicSource(t)
			publicSource.EXPECT().
				SearchKeys(mock.Anything, "test-usage").
				Return([]*jwa.JWK{publicKey}, nil).
				Once()

			recipients, err := core.NewJwkRecipients(publicSource, keys)
			require.NoError(t, err)

			claims := &testClaims{Foo: "bar"}
			signer := core.NewClaimsSign(producers, keys)
			verifier := core.NewClaimsVerify[testClaims](recipients, keys)

			signedClaims, err := signer.Exec(t.Context(), &core.ClaimsSignRequest{
				Claims: claims,
				Usage:  "test-usage",
			})
			require.NoError(t, err)

			verifiedClaims, err := verifier.Exec(t.Context(), &core.ClaimsVerifyRequest{
				Token: signedClaims,
				Usage: "test-usage",
			})
			require.NoError(t, err)
			require.Equal(t, claims, verifiedClaims)

			privateSource.AssertExpectations(t)
			publicSource.AssertExpectations(t)
		})
	}
}
