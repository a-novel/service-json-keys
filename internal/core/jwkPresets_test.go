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
	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
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

			keys:      configtest.SupportedAlgorithmKeys(),
			expectLen: len(configtest.SupportedAlgorithms),
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

		keys := configtest.TestKeys(jwa.Ed25519, config.JwkKey{Cache: time.Millisecond, StaleIfError: time.Hour})

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
		freshRequest := &core.ClaimsSignRequest{Claims: &testClaims{Message: "fresh"}, Usage: "test-usage"}
		staleRequest := &core.ClaimsSignRequest{Claims: &testClaims{Message: "stale"}, Usage: "test-usage"}

		_, err = signer.Exec(t.Context(), freshRequest)
		require.NoError(t, err)

		time.Sleep(2 * time.Millisecond)

		_, err = signer.Exec(t.Context(), staleRequest)
		require.NoError(t, err)

		privateSource.AssertExpectations(t)
	})
	t.Run("Success/StaleWhileRevalidate", func(t *testing.T) {
		t.Parallel()

		privateKey, _, err := jwk.GenerateED25519()
		require.NoError(t, err)

		keys := configtest.TestKeys(jwa.Ed25519, config.JwkKey{Cache: time.Millisecond, StaleWhileRevalidate: time.Hour})

		// The refetch blocks until the test releases it. Signing must not wait on it.
		fetching := make(chan struct{})
		release := make(chan struct{})

		privateSource := coremocks.NewMockJwkPrivateSource(t)
		privateSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			Return([]*jwa.JWK{privateKey.JWK}, nil).
			Once()
		privateSource.EXPECT().
			SearchKeys(mock.Anything, "test-usage").
			RunAndReturn(func(_ context.Context, _ string) ([]*jwa.JWK, error) {
				close(fetching)
				<-release

				return []*jwa.JWK{privateKey.JWK}, nil
			}).
			Once()

		producers, err := core.NewJwkProducers(privateSource, keys)
		require.NoError(t, err)

		type testClaims struct {
			Message string `json:"message"`
		}

		signer := core.NewClaimsSign(producers, keys)
		request := &core.ClaimsSignRequest{Claims: &testClaims{Message: "test"}, Usage: "test-usage"}

		_, err = signer.Exec(t.Context(), request)
		require.NoError(t, err)

		time.Sleep(2 * time.Millisecond)

		_, err = signer.Exec(t.Context(), request)
		require.NoError(t, err)

		<-fetching
		close(release)

		privateSource.AssertExpectations(t)
	})
}
