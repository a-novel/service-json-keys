package tokens_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
)

func TestClaimsVerify(t *testing.T) {
	t.Parallel()

	type testClaims struct {
		Foo string `json:"foo"`
	}

	testConfig := configtest.TestKeys(jwa.Ed25519, config.JwkKey{})

	testCases := []struct {
		name string

		request *tokens.ClaimsVerifyRequest

		recipients tokens.JwkRecipients
		keysConfig map[string]*config.Jwk

		expectErr error
	}{
		{
			name: "Error/ConfigNotFound",

			request: &tokens.ClaimsVerifyRequest{
				Token: "some.token.value",
				Usage: "unknown-usage",
			},

			keysConfig: testConfig,
			recipients: make(tokens.JwkRecipients),

			expectErr: jwkconfig.ErrConfigNotFound,
		},
		{
			name: "Error/NoRecipients",

			request: &tokens.ClaimsVerifyRequest{
				Token: "some.token.value",
				Usage: "test-usage",
			},

			keysConfig: testConfig,
			recipients: make(tokens.JwkRecipients),

			expectErr: jwkconfig.ErrConfigNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := tokens.NewClaimsVerify[testClaims](testCase.recipients, testCase.keysConfig)

			claims, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Nil(t, claims)
		})
	}
}
