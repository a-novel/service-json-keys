package tokens_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jwp"

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
	tokensmocks "github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens/mocks"
)

// verifyClaims are the claims the verify tests sign and read back.
type verifyClaims struct {
	Foo string `json:"foo"`
}

func TestClaimsVerify(t *testing.T) {
	t.Parallel()

	privateKey, publicKey, err := jwk.GenerateED25519()
	require.NoError(t, err)

	keys := configtest.TestKeys(jwa.Ed25519, jwkconfig.Key{Cache: time.Hour})
	token := keys["test-usage"].Token

	// An expired token outlives its TTL by more than the leeway.
	expired := token
	expired.TTL = -time.Hour

	// Another issuer signs with the same key.
	foreign := token
	foreign.Issuer = "other-issuer"

	publicSource := tokensmocks.NewMockJwkPublicSource(t)
	publicSource.EXPECT().
		SearchKeys(mock.Anything, "test-usage").
		Return([]*jwa.JWK{publicKey.JWK}, nil).
		Maybe()

	recipients, err := tokens.NewJwkRecipients(publicSource, keys)
	require.NoError(t, err)

	testCases := []struct {
		name string

		request *tokens.ClaimsVerifyRequest

		recipients tokens.JwkRecipients

		expect    *verifyClaims
		expectErr error
	}{
		{
			name: "Success",

			request: &tokens.ClaimsVerifyRequest{Token: signClaims(t, privateKey.JWK, token), Usage: "test-usage"},

			recipients: recipients,

			expect: &verifyClaims{Foo: "bar"},
		},
		{
			// A refresh flow reads the claims of an expired token.
			name: "Success/IgnoreExpired",

			request: &tokens.ClaimsVerifyRequest{
				Token: signClaims(t, privateKey.JWK, expired), Usage: "test-usage", IgnoreExpired: true,
			},

			recipients: recipients,

			expect: &verifyClaims{Foo: "bar"},
		},
		{
			name: "Error/Expired",

			request: &tokens.ClaimsVerifyRequest{Token: signClaims(t, privateKey.JWK, expired), Usage: "test-usage"},

			recipients: recipients,

			expectErr: jwp.ErrInvalidClaims,
		},
		{
			// Ignoring the expiry never lifts the issuer, audience and subject check.
			name: "Error/ForeignIssuer",

			request: &tokens.ClaimsVerifyRequest{
				Token: signClaims(t, privateKey.JWK, foreign), Usage: "test-usage", IgnoreExpired: true,
			},

			recipients: recipients,

			expectErr: jwp.ErrInvalidClaims,
		},
		{
			name: "Error/ConfigNotFound",

			request: &tokens.ClaimsVerifyRequest{Token: "some.token.value", Usage: "unknown-usage"},

			recipients: make(tokens.JwkRecipients),

			expectErr: jwkconfig.ErrConfigNotFound,
		},
		{
			name: "Error/NoRecipients",

			request: &tokens.ClaimsVerifyRequest{Token: "some.token.value", Usage: "test-usage"},

			recipients: make(tokens.JwkRecipients),

			expectErr: jwkconfig.ErrConfigNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := tokens.NewClaimsVerify[verifyClaims](testCase.recipients, keys)

			claims, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, claims)
		})
	}
}

// signClaims signs verifyClaims with privateKey under the "test-usage" usage, stamped with token's
// TTL, issuer, audience and subject.
func signClaims(t *testing.T, privateKey *jwa.JWK, token jwkconfig.Token) string {
	t.Helper()

	keys := configtest.TestKeys(jwa.Ed25519, jwkconfig.Key{Cache: time.Hour})
	keys["test-usage"].Token = token

	privateSource := coremocks.NewMockJwkProducersServiceSearch(t)
	privateSource.EXPECT().
		Exec(mock.Anything, &core.JwkSearchRequest{Usage: "test-usage", Private: true}).
		Return([]*jwa.JWK{privateKey}, nil)

	producers, err := core.NewJwkProducers(privateSource, keys)
	require.NoError(t, err)

	request := &core.ClaimsSignRequest{Claims: &verifyClaims{Foo: "bar"}, Usage: "test-usage"}

	signed, err := core.NewClaimsSign(producers, keys).Exec(t.Context(), request)
	require.NoError(t, err)

	return signed
}
