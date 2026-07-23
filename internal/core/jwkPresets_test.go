package core_test

import (
	"context"
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

func TestNewJwkPrivateSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		keys map[string]*config.Jwk

		expectErr error
	}{
		{
			name: "Success",

			keys: map[string]*config.Jwk{
				"test-usage": {Alg: jwa.EdDSA},
			},
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

			_, err := core.NewJwkPrivateSource(source, testCase.keys)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}

func TestNewJwkPublicSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		keys map[string]*config.Jwk

		expectErr error
	}{
		{
			name: "Success",

			keys: map[string]*config.Jwk{
				"test-usage": {Alg: jwa.EdDSA},
			},
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

			_, err := core.NewJwkPublicSource(source, testCase.keys)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}

// The signer rotates to a key the moment it is published, but a verifier holds its cached set for
// the whole cache duration — so a token signed with a just-rotated key names a kid the verifier does
// not yet have. Without RefreshOnUnknownKeyID the source scans its stale cache, misses, and reports
// the key not found; with it, the unknown kid forces one refetch and the verifier recovers.
func TestNewJwkPublicSourceRefetchesOnUnknownKeyID(t *testing.T) {
	t.Parallel()

	_, key1Public, err := jwk.GenerateED25519()
	require.NoError(t, err)

	_, key2Public, err := jwk.GenerateED25519()
	require.NoError(t, err)

	require.NotEqual(t, key1Public.KID, key2Public.KID, "the two keys must have distinct ids")

	// The published set: only key1 at first, then key1 and the rotated-in key2. A long cache means
	// the normal refresh never refetches within the test, so recovery can only come from the
	// unknown-kid path.
	source := coremocks.NewMockJwkPublicSource(t)

	var calls int

	source.EXPECT().
		SearchKeys(mock.Anything, "test-usage").
		RunAndReturn(func(context.Context, string) ([]*jwa.JWK, error) {
			calls++
			if calls == 1 {
				return []*jwa.JWK{key1Public.JWK}, nil
			}

			return []*jwa.JWK{key1Public.JWK, key2Public.JWK}, nil
		})

	// A long cache so a normal refresh never refetches within the test — recovery can only come
	// from the unknown-kid path. A tiny interval so that path is not rate-limited against the
	// just-primed cache; in production the cache is minutes old by the time a rotated key appears,
	// well past the 10s default.
	sources, err := core.NewJwkPublicSource(source, map[string]*config.Jwk{
		"test-usage": {Alg: jwa.EdDSA, Key: config.JwkKey{Cache: time.Hour, UnknownKeyIDInterval: time.Millisecond}},
	})
	require.NoError(t, err)

	keySource := sources.EdDSA["test-usage"]
	require.NotNil(t, keySource)

	// Prime the cache with the first published set (key1 only).
	cached, err := keySource.List(t.Context())
	require.NoError(t, err)
	require.Len(t, cached, 1)

	// Let the unknown-kid interval elapse, as it has in production by the time a rotation is seen.
	time.Sleep(2 * time.Millisecond)

	// A token names key2, which the cache does not hold. The lookup misses, and RefreshOnUnknownKeyID
	// forces the refetch that surfaces it.
	got, err := keySource.Get(t.Context(), key2Public.KID)
	require.NoError(t, err, "an unknown kid must trigger a refetch, not fail")
	require.Equal(t, key2Public.KID, got.KID)
}
