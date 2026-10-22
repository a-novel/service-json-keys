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

		expect jwk.JwkKey
	}{
		{
			name:  "Success/Auth",
			usage: "auth",
			expect: jwk.JwkKey{
				TTL:                  168 * time.Hour,
				Rotation:             24 * time.Hour,
				Cache:                30 * time.Minute,
				StaleIfError:         24 * time.Hour,
				StaleWhileRevalidate: 30 * time.Minute,
			},
		},
		{
			name:  "Success/AuthRefresh",
			usage: "auth-refresh",
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
		})
	}
}
