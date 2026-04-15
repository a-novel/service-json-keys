package services_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/services"
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

		request *services.ClaimsSignRequest

		producers  services.JwkProducers
		keysConfig map[string]*config.Jwk

		expectErr error
	}{
		{
			name: "Error/ConfigNotFound",

			request: &services.ClaimsSignRequest{
				Claims: &testClaims{Foo: "bar"},
				Usage:  "unknown-usage",
			},

			keysConfig: testConfig,
			producers:  make(services.JwkProducers),

			expectErr: services.ErrConfigNotFound,
		},
		{
			name: "Error/NoProducers",

			request: &services.ClaimsSignRequest{
				Claims: &testClaims{Foo: "bar"},
				Usage:  "test-usage",
			},

			keysConfig: testConfig,
			producers:  make(services.JwkProducers),

			expectErr: services.ErrConfigNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := services.NewClaimsSign(testCase.producers, testCase.keysConfig)

			_, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
