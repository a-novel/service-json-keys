package tokens_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwk"

	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
	tokensmocks "github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens/mocks"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"
)

// proofClaims are the claims the proof tests sign and read back.
type proofClaims struct {
	Entry string `json:"entry"`
}

func TestAPIKeyProofVerify(t *testing.T) {
	t.Parallel()

	const purpose = "test-purpose"

	errFoo := errors.New("foo")
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

	// serving returns a source that answers every read with served.
	serving := func(served *tokens.APIKey) tokens.APIKeySource {
		source := tokensmocks.NewMockAPIKeySource(t)
		source.EXPECT().SelectKey(mock.Anything, mock.Anything).Return(served, nil).Maybe()

		return source
	}

	privateKey, _, err := jwk.GenerateED25519()
	require.NoError(t, err)

	_, ecdsaKey, err := jwk.GenerateECDSA(jwk.ES256)
	require.NoError(t, err)

	// sign returns a proof of signer for an entry, with its claims changed.
	sign := func(signer *servicejsonkeystest.APIKey, changes map[string]any) string {
		claims := signer.ProofClaims(map[string]any{"entry": "test-entry"})

		return signer.Sign(t, servicejsonkeystest.TypProof, changed(claims, changes))
	}

	testCases := []struct {
		name string

		proof string
		// source serves the keys, those of the test when nil.
		source tokens.APIKeySource

		expect    *tokens.APIKeyProof
		expectErr error
	}{
		{
			name:  "Success",
			proof: sign(key, nil),

			expect: &tokens.APIKeyProof{KeyID: key.ID},
		},
		{
			// What a key signed before it was deactivated still verifies.
			name:  "Success/DeactivatedKey",
			proof: sign(deactivated, nil),

			expect: &tokens.APIKeyProof{KeyID: deactivated.ID},
		},
		{
			// A proof never expires.
			name:  "Success/Old",
			proof: sign(key, map[string]any{"iat": now - 10*365*24*3600}),

			expect: &tokens.APIKeyProof{KeyID: key.ID},
		},
		{
			name:  "Error/Malformed",
			proof: "not-a-proof",

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/UnknownKey",
			proof: sign(unknown, nil),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// Another key signs under the ID of the served one.
			name:  "Error/OtherKey",
			proof: sign(signedBy(t, key), nil),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/LeakedKey",
			proof: sign(leaked, nil),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A key made for another purpose signs for this one.
			name:  "Error/KeyOfAnotherPurpose",
			proof: sign(foreign, map[string]any{"aud": purpose}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/OtherPurpose",
			proof: sign(key, map[string]any{"aud": "other-purpose"}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A request token of the same key, with the claims of a proof, proves nothing.
			name: "Error/RequestToken",
			proof: key.Sign(
				t, servicejsonkeystest.TypRequestToken, key.ProofClaims(map[string]any{"entry": "test-entry"}),
			),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/NoIssueDate",
			proof: sign(key, map[string]any{"iat": nil}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/SignedInTheFuture",
			proof: sign(key, map[string]any{"iat": now + 60}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A proof carries no expiry. One that does is held to it.
			name:  "Error/Expired",
			proof: sign(key, map[string]any{"exp": now - 300}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			name:  "Error/ClaimsOfAnotherShape",
			proof: sign(key, map[string]any{"entry": 12}),

			expectErr: jwt.ErrTokenRejected,
		},
		{
			// A key that cannot be read is no fault of the proof.
			name:   "Error/Source",
			proof:  sign(key, nil),
			source: failing,

			expectErr: tokens.ErrAPIKeySource,
		},
		{
			// The source answers with the key of another ID than the one it was asked.
			name:   "Error/SourceAnswersAnotherKey",
			proof:  sign(key, nil),
			source: serving(servedAPIKey(unknown)),

			expectErr: tokens.ErrAPIKeySource,
		},
		{
			// A private key never leaves its owner, so a source that serves one is at fault.
			name:  "Error/SourceServesAPrivateKey",
			proof: sign(key, nil),
			source: serving(&tokens.APIKey{
				ID: key.ID, Purpose: purpose, Key: privateKey.JWK,
			}),

			expectErr: tokens.ErrAPIKeySource,
		},
		{
			name:  "Error/SourceServesAnotherKind",
			proof: sign(key, nil),
			source: serving(&tokens.APIKey{
				ID: key.ID, Purpose: purpose, Key: ecdsaKey.JWK,
			}),

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

			keys, err := tokens.NewAPIKeys(source, purpose)
			require.NoError(t, err)

			var claims proofClaims

			proof, err := tokens.NewAPIKeyProofVerify(keys).Exec(t.Context(), &tokens.APIKeyProofVerifyRequest{
				Proof:  testCase.proof,
				Claims: &claims,
			})
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, proof)

			if testCase.expectErr == nil {
				require.Equal(t, proofClaims{Entry: "test-entry"}, claims)

				return
			}

			// Only a refused proof answers as one, so a caller tells it from a failure of its own.
			if !errors.Is(testCase.expectErr, jwt.ErrTokenRejected) {
				require.NotErrorIs(t, err, jwt.ErrTokenRejected)
			}
		})
	}
}
