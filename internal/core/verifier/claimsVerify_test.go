package verifier_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
	"github.com/a-novel/service-json-keys/v2/internal/core/verifier"
)

func TestClaimsVerify(t *testing.T) {
	t.Parallel()

	type testClaims struct {
		Foo string `json:"foo"`
	}

	keys := map[string]*jwkconfig.Jwk{
		"test-usage": {Alg: jwa.EdDSA},
	}

	testCases := []struct {
		name string

		request    *verifier.ClaimsVerifyRequest
		recipients verifier.Recipients

		expectErr error
	}{
		{
			name: "Error/ConfigNotFound",
			request: &verifier.ClaimsVerifyRequest{
				Token: "some.token.value",
				Usage: "unknown-usage",
			},
			recipients: make(verifier.Recipients),
			expectErr:  verifier.ErrConfigNotFound,
		},
		{
			name: "Error/NoRecipients",
			request: &verifier.ClaimsVerifyRequest{
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

			claimsVerifier := verifier.NewClaimsVerify[testClaims](testCase.recipients, keys)
			claims, err := claimsVerifier.Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Nil(t, claims)
		})
	}
}
