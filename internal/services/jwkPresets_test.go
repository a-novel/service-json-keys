package services_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/services"
	servicesmocks "github.com/a-novel/service-json-keys/v2/internal/services/mocks"
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

			expectErr: services.ErrJwkPresetUnknownAlgorithm,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockJwkPrivateSource(t)

			_, err := services.NewJwkPrivateSource(source, testCase.keys)
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

			expectErr: services.ErrJwkPresetUnknownAlgorithm,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockJwkPublicSource(t)

			_, err := services.NewJwkPublicSource(source, testCase.keys)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
