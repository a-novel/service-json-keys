package verifier_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/jwk"
	"github.com/a-novel/service-json-keys/v2/internal/verifier"
)

func TestClaims(t *testing.T) {
	t.Parallel()

	type testClaims struct {
		Foo string `json:"foo"`
	}

	keys := map[string]*jwkconfig.Jwk{
		"test-usage": {Alg: jwa.EdDSA},
	}

	testCases := []struct {
		name string

		request    *verifier.Request
		recipients verifier.Recipients

		expectErr error
	}{
		{
			name: "Error/ConfigNotFound",
			request: &verifier.Request{
				Token: "some.token.value",
				Usage: "unknown-usage",
			},
			recipients: make(verifier.Recipients),
			expectErr:  verifier.ErrConfigNotFound,
		},
		{
			name: "Error/NoRecipients",
			request: &verifier.Request{
				Token: "some.token.value",
				Usage: "test-usage",
			},
			recipients: make(verifier.Recipients),
			expectErr:  verifier.ErrConfigNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			claimsVerifier := verifier.NewClaims[testClaims](testCase.recipients, keys)
			claims, err := claimsVerifier.Verify(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Nil(t, claims)
		})
	}
}
