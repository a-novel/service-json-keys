package core_test

import (
	"context"
	"errors"
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

			keys: map[string]*config.Jwk{
				"ed25519": {Alg: jwa.Ed25519},
				"eddsa":   {Alg: jwa.EdDSA}, //nolint:staticcheck // The legacy label is under test.
				"es256":   {Alg: jwa.ES256},
				"es384":   {Alg: jwa.ES384},
				"es512":   {Alg: jwa.ES512},
				"rs256":   {Alg: jwa.RS256},
				"rs384":   {Alg: jwa.RS384},
				"rs512":   {Alg: jwa.RS512},
				"ps256":   {Alg: jwa.PS256},
				"ps384":   {Alg: jwa.PS384},
				"ps512":   {Alg: jwa.PS512},
				"mldsa44": {Alg: jwa.MLDSA44},
				"mldsa65": {Alg: jwa.MLDSA65},
				"mldsa87": {Alg: jwa.MLDSA87},
			},

			expectLen: 14,
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

		keys := map[string]*config.Jwk{
			"test-usage": {
				Alg: jwa.Ed25519,
				Key: config.JwkKey{
					Cache:        time.Millisecond,
					StaleIfError: time.Hour,
				},
				Token: config.JwkToken{
					TTL:      time.Hour,
					Issuer:   "test-issuer",
					Audience: "test-audience",
					Subject:  "test-subject",
				},
			},
		}

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

			keys: map[string]*config.Jwk{
				"ed25519": {Alg: jwa.Ed25519},
				"eddsa":   {Alg: jwa.EdDSA}, //nolint:staticcheck // The legacy label is under test.
				"es256":   {Alg: jwa.ES256},
				"es384":   {Alg: jwa.ES384},
				"es512":   {Alg: jwa.ES512},
				"rs256":   {Alg: jwa.RS256},
				"rs384":   {Alg: jwa.RS384},
				"rs512":   {Alg: jwa.RS512},
				"ps256":   {Alg: jwa.PS256},
				"ps384":   {Alg: jwa.PS384},
				"ps512":   {Alg: jwa.PS512},
				"mldsa44": {Alg: jwa.MLDSA44},
				"mldsa65": {Alg: jwa.MLDSA65},
				"mldsa87": {Alg: jwa.MLDSA87},
			},

			expectLen: 14,
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

		keys := map[string]*config.Jwk{
			"test-usage": {
				Alg: jwa.Ed25519,
				Key: config.JwkKey{
					Cache:                time.Hour,
					UnknownKeyIDInterval: time.Millisecond,
				},
				Token: config.JwkToken{
					TTL:      time.Hour,
					Issuer:   "test-issuer",
					Audience: "test-audience",
					Subject:  "test-subject",
				},
			},
		}

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

		keys := map[string]*config.Jwk{
			"test-usage": {
				Alg: jwa.Ed25519,
				Key: config.JwkKey{
					Cache:        time.Millisecond,
					StaleIfError: time.Hour,
				},
				Token: config.JwkToken{
					TTL:      time.Hour,
					Issuer:   "test-issuer",
					Audience: "test-audience",
					Subject:  "test-subject",
				},
			},
		}

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
