package verifier_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/jwk"
	"github.com/a-novel/service-json-keys/v2/internal/verifier"
)

type source struct{}

func (source) SearchKeys(context.Context, string) ([]*jwa.JWK, error) {
	return nil, nil
}

func TestRecipients(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		keys map[string]*jwkconfig.Jwk

		expectErr error
	}{
		{
			name: "Success",
			keys: map[string]*jwkconfig.Jwk{
				"test-usage": {Alg: jwa.EdDSA},
			},
		},
		{
			name: "Error/UnknownAlgorithm",
			keys: map[string]*jwkconfig.Jwk{
				"test-usage": {Alg: jwa.Alg("unknown")},
			},
			expectErr: verifier.ErrPresetUnknownAlgorithm,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recipients, err := verifier.NewRecipients(source{}, testCase.keys)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr == nil {
				require.NotNil(t, recipients["test-usage"])
			}
		})
	}
}
