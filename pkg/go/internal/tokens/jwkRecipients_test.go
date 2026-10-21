package tokens_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
	tokensmocks "github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens/mocks"
)

func TestJwkRecipients(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		keys map[string]*config.Jwk

		expectLen int
		expectErr error
	}{
		{
			name: "Success/SupportedAlgorithms",

			keys:      configtest.SupportedAlgorithmKeys(),
			expectLen: len(configtest.SupportedAlgorithms),
		},
		{
			name: "Error/UnknownAlgorithm",

			keys: map[string]*config.Jwk{
				"test-usage": {Alg: jwa.Alg("unknown-alg")},
			},

			expectErr: jwkconfig.ErrJwkPresetUnknownAlgorithm,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := tokensmocks.NewMockJwkPublicSource(t)

			recipients, err := tokens.NewJwkRecipients(source, testCase.keys)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Len(t, recipients, testCase.expectLen)

			for _, plugins := range recipients {
				require.NotNil(t, plugins)
				require.Len(t, plugins, 1)
			}

			source.AssertExpectations(t)
		})
	}

	t.Run("Success/RefreshUnknownKeyID", func(t *testing.T) {
		t.Parallel()

		key1Private, key1Public, err := jwk.GenerateED25519()
		require.NoError(t, err)

		key2Private, key2Public, err := jwk.GenerateED25519()
		require.NoError(t, err)
		require.NotEqual(t, key1Public.KID, key2Public.KID)

		keys := configtest.TestKeys(jwa.Ed25519, config.JwkKey{Cache: time.Hour, UnknownKeyIDInterval: time.Millisecond})

		privateSource1 := coremocks.NewMockJwkPrivateSource(t)
		privateSource1.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{key1Private.JWK}, nil).
			Once()

		producers1, err := core.NewJwkProducers(privateSource1, keys)
		require.NoError(t, err)

		privateSource2 := coremocks.NewMockJwkPrivateSource(t)
		privateSource2.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{key2Private.JWK}, nil).
			Once()

		producers2, err := core.NewJwkProducers(privateSource2, keys)
		require.NoError(t, err)

		publicSource := tokensmocks.NewMockJwkPublicSource(t)

		var publicFetches int

		publicSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			RunAndReturn(func(context.Context, string) ([]*jwa.JWK, error) {
				publicFetches++
				if publicFetches == 1 {
					return []*jwa.JWK{key1Public.JWK}, nil
				}

				return []*jwa.JWK{key1Public.JWK, key2Public.JWK}, nil
			})

		recipients, err := tokens.NewJwkRecipients(publicSource, keys)
		require.NoError(t, err)

		type testClaims struct {
			Message string `json:"message"`
		}

		signer1 := core.NewClaimsSign(producers1, keys)
		signRequest1 := &core.ClaimsSignRequest{Claims: &testClaims{Message: "first"}, Usage: "test-usage"}

		token1, err := signer1.Exec(t.Context(), signRequest1)
		require.NoError(t, err)

		verifier := tokens.NewClaimsVerify[testClaims](recipients, keys)
		verifyRequest1 := &tokens.ClaimsVerifyRequest{Token: token1, Usage: "test-usage"}

		claims1, err := verifier.Exec(t.Context(), verifyRequest1)
		require.NoError(t, err)
		require.Equal(t, &testClaims{Message: "first"}, claims1)

		time.Sleep(2 * time.Millisecond)

		signer2 := core.NewClaimsSign(producers2, keys)
		signRequest2 := &core.ClaimsSignRequest{Claims: &testClaims{Message: "second"}, Usage: "test-usage"}

		token2, err := signer2.Exec(t.Context(), signRequest2)
		require.NoError(t, err)

		verifyRequest2 := &tokens.ClaimsVerifyRequest{Token: token2, Usage: "test-usage"}

		claims2, err := verifier.Exec(t.Context(), verifyRequest2)
		require.NoError(t, err)
		require.Equal(t, &testClaims{Message: "second"}, claims2)
		require.Equal(t, 2, publicFetches)

		privateSource1.AssertExpectations(t)
		privateSource2.AssertExpectations(t)
		publicSource.AssertExpectations(t)
	})

	t.Run("Success/StaleIfError", func(t *testing.T) {
		t.Parallel()

		errFoo := errors.New("foo")

		privateKey, publicKey, err := jwk.GenerateED25519()
		require.NoError(t, err)

		keys := configtest.TestKeys(jwa.Ed25519, config.JwkKey{Cache: time.Millisecond, StaleIfError: time.Hour})

		// The signer reads its key once for the header and once for the signature. The millisecond
		// cache can expire between the two reads on a loaded runner, which refetches the key.
		privateSource := coremocks.NewMockJwkPrivateSource(t)
		privateSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{privateKey.JWK}, nil)

		producers, err := core.NewJwkProducers(privateSource, keys)
		require.NoError(t, err)

		// The first fetch fills the cache and the refetch fails, as during an outage.
		publicSource := tokensmocks.NewMockJwkPublicSource(t)
		publicSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{publicKey.JWK}, nil).
			Once()
		publicSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return(nil, errFoo).
			Once()

		recipients, err := tokens.NewJwkRecipients(publicSource, keys)
		require.NoError(t, err)

		type testClaims struct {
			Message string `json:"message"`
		}

		signRequest := &core.ClaimsSignRequest{Claims: &testClaims{Message: "test"}, Usage: "test-usage"}

		token, err := core.NewClaimsSign(producers, keys).Exec(t.Context(), signRequest)
		require.NoError(t, err)

		verifier := tokens.NewClaimsVerify[testClaims](recipients, keys)
		verifyRequest := &tokens.ClaimsVerifyRequest{Token: token, Usage: "test-usage"}

		claims, err := verifier.Exec(t.Context(), verifyRequest)
		require.NoError(t, err)
		require.Equal(t, &testClaims{Message: "test"}, claims)

		time.Sleep(2 * time.Millisecond)

		claims, err = verifier.Exec(t.Context(), verifyRequest)
		require.NoError(t, err)
		require.Equal(t, &testClaims{Message: "test"}, claims)

		privateSource.AssertExpectations(t)
		publicSource.AssertExpectations(t)
	})
}

// loadGeneratedJWK decodes a generated key into the JSON Web Key a key source serves.
func loadGeneratedJWK(t *testing.T, key any) *jwa.JWK {
	t.Helper()

	encoded, err := json.Marshal(key)
	require.NoError(t, err)

	var output jwa.JWK
	require.NoError(t, json.Unmarshal(encoded, &output))

	return &output
}

func TestJwkGenerators(t *testing.T) {
	t.Parallel()

	type testClaims struct {
		Foo string `json:"foo"`
	}

	for _, alg := range configtest.SupportedAlgorithms {
		t.Run("Success/"+string(alg), func(t *testing.T) {
			t.Parallel()

			generator, ok := core.JwkGenerators[alg]
			require.True(t, ok)

			generatedKey, err := generator()
			require.NoError(t, err)

			privateKey := loadGeneratedJWK(t, generatedKey.PrivateKey)
			publicKey := loadGeneratedJWK(t, generatedKey.PublicKey)
			require.Equal(t, generatedKey.PrivateKID, privateKey.KID)
			require.Equal(t, generatedKey.PublicKID, publicKey.KID)

			keys := configtest.TestKeys(alg, config.JwkKey{Cache: 30 * time.Minute})

			privateSource := coremocks.NewMockJwkPrivateSource(t)
			privateSource.EXPECT().
				SearchKeys(mock.Anything, "test-usage").
				Return([]*jwa.JWK{privateKey}, nil).
				Once()

			producers, err := core.NewJwkProducers(privateSource, keys)
			require.NoError(t, err)

			publicSource := tokensmocks.NewMockJwkPublicSource(t)
			publicSource.EXPECT().
				SearchKeys(mock.Anything, "test-usage").
				Return([]*jwa.JWK{publicKey}, nil).
				Once()

			recipients, err := tokens.NewJwkRecipients(publicSource, keys)
			require.NoError(t, err)

			claims := &testClaims{Foo: "bar"}
			signer := core.NewClaimsSign(producers, keys)
			verifier := tokens.NewClaimsVerify[testClaims](recipients, keys)

			signRequest := &core.ClaimsSignRequest{Claims: claims, Usage: "test-usage"}

			signedClaims, err := signer.Exec(t.Context(), signRequest)
			require.NoError(t, err)

			verifyRequest := &tokens.ClaimsVerifyRequest{Token: signedClaims, Usage: "test-usage"}

			verifiedClaims, err := verifier.Exec(t.Context(), verifyRequest)
			require.NoError(t, err)
			require.Equal(t, claims, verifiedClaims)

			// Both the published key and the token carry the configured label, which is what a
			// verifier pinned to that label checks.
			var header struct {
				Alg jwa.Alg `json:"alg"`
			}

			encodedHeader, _, _ := strings.Cut(signedClaims, ".")
			decodedHeader, err := base64.RawURLEncoding.DecodeString(encodedHeader)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(decodedHeader, &header))
			require.Equal(t, alg, header.Alg)
			require.Equal(t, alg, publicKey.Alg)

			privateSource.AssertExpectations(t)
			publicSource.AssertExpectations(t)
		})
	}
}
