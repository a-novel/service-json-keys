package tokens_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2"

	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
	tokensmocks "github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens/mocks"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"
)

func TestAPIKeyRequestVerify(t *testing.T) {
	t.Parallel()

	const purpose = "test-purpose"

	errFoo := errors.New("foo")
	body := []byte(`{"hello":"world"}`)
	now := time.Now().Unix()

	key := servicejsonkeystest.NewAPIKey(t, purpose)
	unknown := servicejsonkeystest.NewAPIKey(t, purpose)
	foreign := servicejsonkeystest.NewAPIKey(t, "other-purpose")

	deactivated := servicejsonkeystest.NewAPIKey(t, purpose)
	deactivated.Deactivated = true

	leaked := servicejsonkeystest.NewAPIKey(t, purpose)
	leaked.Leaked = true

	failing := tokensmocks.NewMockAPIKeySource(t)
	failing.EXPECT().SelectKey(mock.Anything, mock.Anything).Return(nil, errFoo).Maybe()

	// sign returns a request token of signer for the body, with its claims changed.
	sign := func(signer *servicejsonkeystest.APIKey, changes map[string]any) string {
		return signer.Sign(t, servicejsonkeystest.TypRequestToken, changed(signer.RequestClaims(body), changes))
	}

	testCases := []struct {
		name string

		token string
		// body is the body of the request, the one the tokens are signed for when nil.
		body io.Reader
		// source serves the keys, those of the test when nil.
		source tokens.APIKeySource

		expect    *tokens.APIKeyRequest
		expectErr error
	}{
		{
			name:  "Success",
			token: sign(key, nil),

			expect: &tokens.APIKeyRequest{KeyID: key.ID, Body: body},
		},
		{
			// The owner's clock runs ahead of the verifier's, within the leeway.
			name:  "Success/SignedAhead",
			token: sign(key, map[string]any{"iat": now + 20, "exp": now + 320}),

			expect: &tokens.APIKeyRequest{KeyID: key.ID, Body: body},
		},
		{
			name:  "Error/Malformed",
			token: "not-a-token",

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/NoToken",
			token: "",

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/UnknownKey",
			token: sign(unknown, nil),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// Another key signs under the ID of the served one.
			name:  "Error/OtherKey",
			token: sign(signedBy(t, key), nil),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/DeactivatedKey",
			token: sign(deactivated, nil),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/LeakedKey",
			token: sign(leaked, nil),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A key made for another purpose signs for this one.
			name:  "Error/KeyOfAnotherPurpose",
			token: sign(foreign, map[string]any{"aud": purpose}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/OtherPurpose",
			token: sign(key, map[string]any{"aud": "other-purpose"}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/NoPurpose",
			token: sign(key, map[string]any{"aud": nil}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A token names no issuer: its key is its only origin.
			name:  "Error/Issuer",
			token: sign(key, map[string]any{"iss": "someone"}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A proof of the same key, with the claims of a request token, authenticates nothing.
			name:  "Error/Proof",
			token: key.Sign(t, servicejsonkeystest.TypProof, key.RequestClaims(body)),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/NoType",
			token: key.Sign(t, "", key.RequestClaims(body)),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/Expired",
			token: sign(key, map[string]any{"iat": now - 600, "exp": now - 300}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/NoExpiry",
			token: sign(key, map[string]any{"exp": nil}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/NoIssueDate",
			token: sign(key, map[string]any{"iat": nil}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/LivesTooLong",
			token: sign(key, map[string]any{"iat": now, "exp": now + 301}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/ExpiresBeforeSigned",
			token: sign(key, map[string]any{"iat": now + 20, "exp": now + 10}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/SignedInTheFuture",
			token: sign(key, map[string]any{"iat": now + 60, "exp": now + 120}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/OtherBody",
			token: sign(key, nil),
			body:  strings.NewReader(`{"hello":"you"}`),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/NoDigest",
			token: sign(key, map[string]any{"bodySHA256": nil}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/DigestNotAString",
			token: sign(key, map[string]any{"bodySHA256": 12}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name: "Error/UppercaseDigest",
			token: sign(key, map[string]any{
				"bodySHA256": strings.ToUpper(fmt.Sprint(key.RequestClaims(body)["bodySHA256"])),
			}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A refused token ends the verification before the body is read.
			name:  "Error/RefusedBeforeBody",
			token: sign(unknown, nil),
			body:  iotest.ErrReader(errFoo),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/UnreadableBody",
			token: sign(key, nil),
			body:  iotest.ErrReader(errFoo),

			expectErr: tokens.ErrRequestBody,
		},
		{
			// A key that cannot be read is no fault of the token.
			name:   "Error/Source",
			token:  sign(key, nil),
			source: failing,

			expectErr: tokens.ErrAPIKeySource,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := testCase.source
			if source == nil {
				source = newAPIKeySource(t, key, deactivated, leaked, foreign)
			}

			requestBody := testCase.body
			if requestBody == nil {
				requestBody = bytes.NewReader(body)
			}

			keys, err := tokens.NewAPIKeys(source, purpose)
			require.NoError(t, err)

			request, err := tokens.NewAPIKeyRequestVerify(keys).Exec(t.Context(), &tokens.APIKeyRequestVerifyRequest{
				Token: testCase.token,
				Body:  requestBody,
			})
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, request)

			// Only a refused token answers as one, so a caller tells it from a failure of its own.
			if testCase.expectErr != nil && !errors.Is(testCase.expectErr, jwt.ErrTokenRejected) {
				require.NotErrorIs(t, err, jwt.ErrTokenRejected)
			}
		})
	}
}
