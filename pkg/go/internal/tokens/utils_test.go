package tokens_test

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
	tokensmocks "github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens/mocks"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"
)

// newAPIKeySource returns a source that serves keys, and answers any other ID as unknown.
func newAPIKeySource(t *testing.T, keys ...*servicejsonkeystest.APIKey) *tokensmocks.MockAPIKeySource {
	t.Helper()

	source := tokensmocks.NewMockAPIKeySource(t)
	source.EXPECT().
		SelectKey(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, id string) (*tokens.APIKey, error) {
			for _, key := range keys {
				if key.ID == id {
					return servedAPIKey(key), nil
				}
			}

			return nil, tokens.ErrAPIKeyNotFound
		}).
		Maybe()

	return source
}

// servedAPIKey returns a test key as a source serves it.
func servedAPIKey(key *servicejsonkeystest.APIKey) *tokens.APIKey {
	return &tokens.APIKey{
		ID:          key.ID,
		Purpose:     key.Purpose,
		Key:         key.PublicKey,
		Deactivated: key.Deactivated,
		Leaked:      key.Leaked,
	}
}

// changed returns claims with each entry of changes set, and each nil one removed.
func changed(claims, changes map[string]any) map[string]any {
	maps.Copy(claims, changes)
	maps.DeleteFunc(claims, func(_ string, value any) bool { return value == nil })

	return claims
}

// signedBy returns key as another key would forge it: under the same ID, with a private key of its
// own.
func signedBy(t *testing.T, key *servicejsonkeystest.APIKey) *servicejsonkeystest.APIKey {
	t.Helper()

	forged := servicejsonkeystest.NewAPIKey(t, key.Purpose)
	forged.ID = key.ID

	return forged
}
