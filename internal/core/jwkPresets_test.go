package core_test

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
	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
)

func TestJwkProducers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		keys map[string]*config.Jwk

		expectLen int
		expectErr error
	}{
		{
			name: "Success/SupportedAlgorithms",

			keys:      supportedAlgorithmKeys(),
			expectLen: len(supportedAlgorithms),
		},
		{
			name: "Error/UnknownAlgorithm",

			keys: map[string]*config.Jwk{
				"test-usage": {Alg: jwa.Alg("unknown-alg")},
			},

			expectErr: core.ErrJwkPresetUnknownAlgorithm,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := coremocks.NewMockJwkPrivateSource(t)

			producers, err := core.NewJwkProducers(source, testCase.keys)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Len(t, producers, testCase.expectLen)

			for _, plugins := range producers {
				require.Len(t, plugins, 1)
			}

			source.AssertExpectations(t)
		})
	}

	t.Run("Success/StaleIfError", func(t *testing.T) {
		t.Parallel()

		errFoo := errors.New("foo")

		privateKey, _, err := jwk.GenerateED25519()
		require.NoError(t, err)

		keys := testKeys(jwa.Ed25519, config.JwkKey{Cache: time.Millisecond, StaleIfError: time.Hour})

		// The first fetch fills the cache and the refetch fails, as during an outage.
		privateSource := coremocks.NewMockJwkPrivateSource(t)
		privateSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{privateKey.JWK}, nil).
			Once()
		privateSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return(nil, errFoo).
			Once()

		producers, err := core.NewJwkProducers(privateSource, keys)
		require.NoError(t, err)

		type testClaims struct {
			Message string `json:"message"`
		}

		signer := core.NewClaimsSign(producers, keys)

		_, err = signer.Exec(t.Context(), &core.ClaimsSignRequest{
			Claims: &testClaims{Message: "fresh"},
			Usage:  "test-usage",
		})
		require.NoError(t, err)

		time.Sleep(2 * time.Millisecond)

		_, err = signer.Exec(t.Context(), &core.ClaimsSignRequest{
			Claims: &testClaims{Message: "stale"},
			Usage:  "test-usage",
		})
		require.NoError(t, err)

		privateSource.AssertExpectations(t)
	})
}

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

			keys:      supportedAlgorithmKeys(),
			expectLen: len(supportedAlgorithms),
		},
		{
			name: "Error/UnknownAlgorithm",

			keys: map[string]*config.Jwk{
				"test-usage": {Alg: jwa.Alg("unknown-alg")},
			},

			expectErr: core.ErrJwkPresetUnknownAlgorithm,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := coremocks.NewMockJwkPublicSource(t)

			recipients, err := core.NewJwkRecipients(source, testCase.keys)
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

		keys := testKeys(jwa.Ed25519, config.JwkKey{Cache: time.Hour, UnknownKeyIDInterval: time.Millisecond})

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

		publicSource := coremocks.NewMockJwkPublicSource(t)

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

		recipients, err := core.NewJwkRecipients(publicSource, keys)
		require.NoError(t, err)

		type testClaims struct {
			Message string `json:"message"`
		}

		signer1 := core.NewClaimsSign(producers1, keys)
		token1, err := signer1.Exec(t.Context(), &core.ClaimsSignRequest{
			Claims: &testClaims{Message: "first"},
			Usage:  "test-usage",
		})
		require.NoError(t, err)

		verifier := core.NewClaimsVerify[testClaims](recipients, keys)
		claims1, err := verifier.Exec(t.Context(), &core.ClaimsVerifyRequest{
			Token: token1,
			Usage: "test-usage",
		})
		require.NoError(t, err)
		require.Equal(t, &testClaims{Message: "first"}, claims1)

		time.Sleep(2 * time.Millisecond)

		signer2 := core.NewClaimsSign(producers2, keys)
		token2, err := signer2.Exec(t.Context(), &core.ClaimsSignRequest{
			Claims: &testClaims{Message: "second"},
			Usage:  "test-usage",
		})
		require.NoError(t, err)

		claims2, err := verifier.Exec(t.Context(), &core.ClaimsVerifyRequest{
			Token: token2,
			Usage: "test-usage",
		})
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

		keys := testKeys(jwa.Ed25519, config.JwkKey{Cache: time.Millisecond, StaleIfError: time.Hour})

		// The signer reads its key once for the header and once for the signature. The millisecond
		// cache can expire between the two reads on a loaded runner, which refetches the key.
		privateSource := coremocks.NewMockJwkPrivateSource(t)
		privateSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{privateKey.JWK}, nil)

		producers, err := core.NewJwkProducers(privateSource, keys)
		require.NoError(t, err)

		// The first fetch fills the cache and the refetch fails, as during an outage.
		publicSource := coremocks.NewMockJwkPublicSource(t)
		publicSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{publicKey.JWK}, nil).
			Once()
		publicSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return(nil, errFoo).
			Once()

		recipients, err := core.NewJwkRecipients(publicSource, keys)
		require.NoError(t, err)

		type testClaims struct {
			Message string `json:"message"`
		}

		token, err := core.NewClaimsSign(producers, keys).Exec(t.Context(), &core.ClaimsSignRequest{
			Claims: &testClaims{Message: "test"},
			Usage:  "test-usage",
		})
		require.NoError(t, err)

		verifier := core.NewClaimsVerify[testClaims](recipients, keys)

		claims, err := verifier.Exec(t.Context(), &core.ClaimsVerifyRequest{
			Token: token,
			Usage: "test-usage",
		})
		require.NoError(t, err)
		require.Equal(t, &testClaims{Message: "test"}, claims)

		time.Sleep(2 * time.Millisecond)

		claims, err = verifier.Exec(t.Context(), &core.ClaimsVerifyRequest{
			Token: token,
			Usage: "test-usage",
		})
		require.NoError(t, err)
		require.Equal(t, &testClaims{Message: "test"}, claims)

		privateSource.AssertExpectations(t)
		publicSource.AssertExpectations(t)
	})
}

// supportedAlgorithmKeys configures one usage per supported algorithm, named after it.
func supportedAlgorithmKeys() map[string]*config.Jwk {
	keys := make(map[string]*config.Jwk, len(supportedAlgorithms))
	for _, alg := range supportedAlgorithms {
		keys[string(alg)] = &config.Jwk{Alg: alg}
	}

	return keys
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

	for _, alg := range supportedAlgorithms {
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

			keys := testKeys(alg, config.JwkKey{Cache: 30 * time.Minute})

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
