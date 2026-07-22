package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
)

func TestClaimsSign(t *testing.T) {
	t.Parallel()

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

		request *core.ClaimsSignRequest

		producers  core.JwkProducers
		keysConfig map[string]*config.Jwk

		expectErr error
	}{
		{
			name: "Error/ConfigNotFound",

			request: &core.ClaimsSignRequest{
				Claims: &testClaims{Foo: "bar"},
				Usage:  "unknown-usage",
			},

			keysConfig: testConfig,
			producers:  make(core.JwkProducers),

			expectErr: core.ErrConfigNotFound,
		},
		{
			name: "Error/NoProducers",

			request: &core.ClaimsSignRequest{
				Claims: &testClaims{Foo: "bar"},
				Usage:  "test-usage",
			},

			keysConfig: testConfig,
			producers:  make(core.JwkProducers),

			expectErr: core.ErrConfigNotFound,
		},
		{
			// The envelope's own claims are stamped from the usage config, so a
			// caller naming one is refused when the token is encoded. A usage
			// registered with no plugins still reaches that point, which is where
			// the claims are serialized.
			name: "Error/ReservedClaim",

			request: &core.ClaimsSignRequest{
				Claims: map[string]any{"sub": "attacker", "foo": "bar"},
				Usage:  "test-usage",
			},

			keysConfig: testConfig,
			producers:  core.JwkProducers{"test-usage": nil},

			expectErr: core.ErrReservedClaim,
		},
		{
			name: "Success/UnreservedClaims",

			request: &core.ClaimsSignRequest{
				Claims: &testClaims{Foo: "bar"},
				Usage:  "test-usage",
			},

			keysConfig: testConfig,
			producers:  core.JwkProducers{"test-usage": nil},

			expectErr: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := core.NewClaimsSign(testCase.producers, testCase.keysConfig)

			_, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
