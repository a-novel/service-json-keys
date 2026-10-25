package jwk_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/config/jwk"
)

func TestJwkKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		usage string
		// verifiedFor names the usage whose token lifetime a key must outlast after its last
		// signature: the access token's signature is re-verified until its refresh token expires.
		verifiedFor string

		expect jwk.JwkKey
	}{
		{
			name:        "Success/Auth",
			usage:       "auth",
			verifiedFor: "auth-refresh",
			expect: jwk.JwkKey{
				TTL:                  194 * time.Hour,
				Rotation:             24 * time.Hour,
				Cache:                30 * time.Minute,
				StaleIfError:         24 * time.Hour,
				StaleWhileRevalidate: 30 * time.Minute,
			},
		},
		{
			name:        "Success/AuthRefresh",
			usage:       "auth-refresh",
			verifiedFor: "auth-refresh",
			expect: jwk.JwkKey{
				TTL:                  720 * time.Hour,
				Rotation:             168 * time.Hour,
				Cache:                30 * time.Minute,
				StaleIfError:         24 * time.Hour,
				StaleWhileRevalidate: 30 * time.Minute,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			usage, ok := jwk.PresetDefault[testCase.usage]
			require.True(t, ok)
			require.Equal(t, testCase.expect, usage.Key)

			// A key signs until the next rotation reaches every producer cache.
			verified, ok := jwk.PresetDefault[testCase.verifiedFor]
			require.True(t, ok)
			require.GreaterOrEqual(t,
				usage.Key.TTL,
				usage.Key.Rotation+usage.Key.Cache+verified.Token.TTL+verified.Token.Leeway,
			)
		})
	}
}

func TestJwkToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		usage string

		expect jwk.JwkToken
	}{
		{
			name:  "Success/Auth",
			usage: "auth",
			expect: jwk.JwkToken{
				TTL:      time.Hour,
				Issuer:   "anovel-authentication",
				Audience: "anovel-client",
				Subject:  "user-access",
				Leeway:   5 * time.Minute,
			},
		},
		{
			name:  "Success/AuthRefresh",
			usage: "auth-refresh",
			expect: jwk.JwkToken{
				TTL:      168 * time.Hour,
				Issuer:   "anovel-authentication",
				Audience: "anovel-client",
				Subject:  "user-refresh",
				Leeway:   5 * time.Minute,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			usage, ok := jwk.PresetDefault[testCase.usage]
			require.True(t, ok)
			require.Equal(t, testCase.expect, usage.Token)
		})
	}
}
