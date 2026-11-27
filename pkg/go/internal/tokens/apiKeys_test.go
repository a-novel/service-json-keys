package tokens_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2"

	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
	tokensmocks "github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens/mocks"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"
)

func TestNewAPIKeys(t *testing.T) {
	t.Parallel()

	const purpose = "test-purpose"

	t.Run("Success/KeepsAKey", func(t *testing.T) {
		t.Parallel()

		key := servicejsonkeystest.NewAPIKey(t, purpose)

		// The second verification finds the key the first one read.
		source := tokensmocks.NewMockAPIKeySource(t)
		source.EXPECT().SelectKey(mock.Anything, key.ID).Return(servedAPIKey(key), nil).Once()

		keys, err := tokens.NewAPIKeys(source, purpose)
		require.NoError(t, err)

		service := tokens.NewAPIKeyProofVerify(keys)

		for range 2 {
			_, err = service.Exec(t.Context(), &tokens.APIKeyProofVerifyRequest{
				Proof:  key.Sign(t, servicejsonkeystest.TypProof, key.ProofClaims(nil)),
				Claims: &struct{}{},
			})
			require.NoError(t, err)
		}
	})

	t.Run("Success/ForgetsAKeyItRefuses", func(t *testing.T) {
		t.Parallel()

		key := servicejsonkeystest.NewAPIKey(t, "other-purpose")

		// A key made for another purpose is read again, since its owner may hold one made for this.
		source := tokensmocks.NewMockAPIKeySource(t)
		source.EXPECT().SelectKey(mock.Anything, key.ID).Return(servedAPIKey(key), nil).Twice()

		keys, err := tokens.NewAPIKeys(source, purpose)
		require.NoError(t, err)

		service := tokens.NewAPIKeyProofVerify(keys)

		for range 2 {
			_, err = service.Exec(t.Context(), &tokens.APIKeyProofVerifyRequest{
				Proof:  key.Sign(t, servicejsonkeystest.TypProof, key.ProofClaims(nil)),
				Claims: &struct{}{},
			})
			require.ErrorIs(t, err, jwt.ErrTokenRejected)
		}
	})

	t.Run("Success/StartsOverWhenFull", func(t *testing.T) {
		t.Parallel()

		key := servicejsonkeystest.NewAPIKey(t, purpose)

		// The source serves the key under any ID, and counts the reads of each.
		reads := make(map[string]int)

		source := tokensmocks.NewMockAPIKeySource(t)
		source.EXPECT().
			SelectKey(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, id string) (*tokens.APIKey, error) {
				reads[id]++

				served := servedAPIKey(key)
				served.ID = id

				return served, nil
			})

		keys, err := tokens.NewAPIKeys(source, purpose)
		require.NoError(t, err)

		service := tokens.NewAPIKeyProofVerify(keys)

		// verify verifies a proof that names the key under id.
		verify := func(id string) {
			named := *key
			named.ID = id

			_, err := service.Exec(t.Context(), &tokens.APIKeyProofVerifyRequest{
				Proof:  named.Sign(t, servicejsonkeystest.TypProof, named.ProofClaims(nil)),
				Claims: &struct{}{},
			})
			require.NoError(t, err)
		}

		verify("first")

		// A caller names as many keys as the cache holds, which then forgets the first one.
		for i := range tokens.APIKeyCacheSize {
			verify(fmt.Sprintf("key-%d", i))
		}

		verify("first")
		require.Equal(t, 2, reads["first"])
	})

	t.Run("Error/PurposeRequired", func(t *testing.T) {
		t.Parallel()

		_, err := tokens.NewAPIKeys(tokensmocks.NewMockAPIKeySource(t), "")
		require.ErrorIs(t, err, tokens.ErrPurposeRequired)
	})
}
