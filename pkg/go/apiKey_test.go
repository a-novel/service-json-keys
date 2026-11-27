package servicejsonkeys_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"
)

// entryClaims are the claims the proof tests sign and read back.
type entryClaims struct {
	Entry string `json:"entry"`
}

func TestAPIKeyVerifier(t *testing.T) {
	t.Parallel()

	const (
		purpose = servicejsonkeys.APIKeyPurposePublishNarrativeEnginePlugin
		body    = `{"plugin":"scenes-composer","version":"1.0.0"}`
	)

	key := servicejsonkeystest.NewAPIKey(t, purpose)
	unknown := servicejsonkeystest.NewAPIKey(t, purpose)

	verifier, err := servicejsonkeys.NewAPIKeyVerifier(servicejsonkeystest.APIKeys{key}, purpose)
	require.NoError(t, err)

	t.Run("Success/RequestToken", func(t *testing.T) {
		t.Parallel()

		request, err := verifier.VerifyRequestToken(t.Context(), &servicejsonkeys.VerifyRequestTokenRequest{
			Token: key.Sign(t, servicejsonkeystest.TypRequestToken, key.RequestClaims([]byte(body))),
			Body:  strings.NewReader(body),
		})
		require.NoError(t, err)
		require.Equal(t, &servicejsonkeys.APIKeyRequest{KeyID: key.ID, Body: []byte(body)}, request)
	})

	t.Run("Success/Proof", func(t *testing.T) {
		t.Parallel()

		var claims entryClaims

		proof, err := verifier.VerifyProof(t.Context(), &servicejsonkeys.VerifyProofRequest{
			Proof:  key.Sign(t, servicejsonkeystest.TypProof, key.ProofClaims(map[string]any{"entry": "test-entry"})),
			Claims: &claims,
		})
		require.NoError(t, err)
		require.Equal(t, &servicejsonkeys.APIKeyProof{KeyID: key.ID}, proof)
		require.Equal(t, entryClaims{Entry: "test-entry"}, claims)
	})

	t.Run("Error/UnknownKey", func(t *testing.T) {
		t.Parallel()

		// The service answers NotFound, which refuses the token.
		_, err := verifier.VerifyProof(t.Context(), &servicejsonkeys.VerifyProofRequest{
			Proof:  unknown.Sign(t, servicejsonkeystest.TypProof, unknown.ProofClaims(nil)),
			Claims: &entryClaims{},
		})
		require.ErrorIs(t, err, servicejsonkeys.ErrTokenRejected)
	})

	t.Run("Error/KeyUnavailable", func(t *testing.T) {
		t.Parallel()

		down, err := servicejsonkeys.NewAPIKeyVerifier(unavailableAPIKeys{}, purpose)
		require.NoError(t, err)

		// A key the service could not serve refuses nothing: the caller answers with a fault of its own.
		_, err = down.VerifyProof(t.Context(), &servicejsonkeys.VerifyProofRequest{
			Proof:  key.Sign(t, servicejsonkeystest.TypProof, key.ProofClaims(nil)),
			Claims: &entryClaims{},
		})
		require.Error(t, err)
		require.NotErrorIs(t, err, servicejsonkeys.ErrTokenRejected)
	})

	t.Run("Error/PurposeRequired", func(t *testing.T) {
		t.Parallel()

		_, err := servicejsonkeys.NewAPIKeyVerifier(servicejsonkeystest.APIKeys{key}, "")
		require.Error(t, err)
	})

	t.Run("Error/RemoteKey", func(t *testing.T) {
		t.Parallel()

		client, err := servicejsonkeys.NewClient(configtest.GrpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)

		defer client.Close()

		remote, err := servicejsonkeys.NewAPIKeyVerifier(client, purpose)
		require.NoError(t, err)

		// The stack holds a key nothing signs for. A token that names it is read against the key the
		// service serves, which refuses the signature of another one.
		forged := servicejsonkeystest.NewAPIKey(t, purpose)
		forged.ID = configtest.APIKeyID

		_, err = remote.VerifyProof(t.Context(), &servicejsonkeys.VerifyProofRequest{
			Proof:  forged.Sign(t, servicejsonkeystest.TypProof, forged.ProofClaims(nil)),
			Claims: &entryClaims{},
		})
		require.ErrorIs(t, err, servicejsonkeys.ErrTokenRejected)
		require.ErrorContains(t, err, "invalid signature")
	})
}
