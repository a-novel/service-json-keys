package core_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
)

func TestClaimsVerify(t *testing.T) {
	t.Parallel()

	type testClaims struct {
		Foo string `json:"foo"`
	}

	testConfig := testKeys(jwa.Ed25519, config.JwkKey{})

	testCases := []struct {
		name string

		request *core.ClaimsVerifyRequest

		recipients core.JwkRecipients
		keysConfig map[string]*config.Jwk

		expectErr error
	}{
		{
			name: "Error/ConfigNotFound",

			request: &core.ClaimsVerifyRequest{
				Token: "some.token.value",
				Usage: "unknown-usage",
			},

			keysConfig: testConfig,
			recipients: make(core.JwkRecipients),

			expectErr: core.ErrConfigNotFound,
		},
		{
			name: "Error/NoRecipients",

			request: &core.ClaimsVerifyRequest{
				Token: "some.token.value",
				Usage: "test-usage",
			},

			keysConfig: testConfig,
			recipients: make(core.JwkRecipients),

			expectErr: core.ErrConfigNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := core.NewClaimsVerify[testClaims](testCase.recipients, testCase.keysConfig)

			claims, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Nil(t, claims)
		})
	}
}
