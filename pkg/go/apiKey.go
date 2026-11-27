package servicejsonkeys

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/a-novel-kit/jwt/v2"

	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
)

// APIKeyPurpose names the one thing an API key may do. The service that owns a purpose verifies
// what the keys made for it signed.
type APIKeyPurpose = string

// APIKeyPurposePublishNarrativeEnginePlugin is the purpose of the keys that publish a version of a
// Narrative Engine plugin.
//
//nolint:gosec // A purpose names what a key may do, and holds no credential.
const APIKeyPurposePublishNarrativeEnginePlugin APIKeyPurpose = "publish-narrative-engine-plugin"

const (
	// APIKeyRequestLifetime is the longest a request token may live: its "exp" claim comes that long
	// after its "iat" claim at most.
	APIKeyRequestLifetime = tokens.APIKeyRequestLifetime
	// APIKeyLeeway is the clock skew a verifier tolerates between the owner of an API key and itself.
	APIKeyLeeway = tokens.APIKeyLeeway
)

// ErrTokenRejected is the error a refused token matches: a request token or a proof no usable API
// key vouches for. Any other error of an [APIKeyVerifier] is a key or a body that could not be read.
var ErrTokenRejected = jwt.ErrTokenRejected

type (
	// VerifyRequestTokenRequest holds the parameters for an [APIKeyVerifier.VerifyRequestToken] call.
	VerifyRequestTokenRequest = tokens.APIKeyRequestVerifyRequest
	// APIKeyRequest is a request an API key authenticated: the ID of the key, and the body its token
	// was signed for.
	APIKeyRequest = tokens.APIKeyRequest
	// VerifyProofRequest holds the parameters for an [APIKeyVerifier.VerifyProof] call.
	VerifyProofRequest = tokens.APIKeyProofVerifyRequest
	// APIKeyProof is a proof an API key signed: the ID of that key.
	APIKeyProof = tokens.APIKeyProof
)

// APIKeyReader reads an API key by its ID. A [BaseClient] is one.
type APIKeyReader interface {
	APIKeyGet(ctx context.Context, req *APIKeyGetRequest, opts ...grpc.CallOption) (*APIKeyGetResponse, error)
}

// An APIKeyVerifier verifies what the API keys of one purpose signed. It reads the key a token
// names from the service and keeps it for a few minutes, so a deactivated key authenticates
// requests that long. Obtain one with [NewAPIKeyVerifier].
//
// Both tokens are JWTs signed with Ed25519, whose header names the key in "kid" and whose "aud"
// claim is the purpose. A verifier checks the "typ" header exactly, so neither passes for the other.
type APIKeyVerifier interface {
	// VerifyRequestToken authenticates a request by its request token, of type "JWT": signed by an
	// active key, carrying "iat" and an "exp" at most [APIKeyRequestLifetime] later, and the SHA-256
	// of the body in "bodySHA256". It reads the body once the rest holds, and returns it.
	//
	// A request token can be replayed until it expires, with the same body alone. The request a key
	// authenticates must be safe to repeat.
	VerifyRequestToken(ctx context.Context, req *VerifyRequestTokenRequest) (*APIKeyRequest, error)
	// VerifyProof verifies a proof, of type "proof+jwt", and decodes its claims. A proof never
	// expires, and one signed by a key deactivated since still verifies. The caller compares the
	// claims with the entry the proof travels with.
	VerifyProof(ctx context.Context, req *VerifyProofRequest) (*APIKeyProof, error)
}

type apiKeyVerifier struct {
	request *tokens.APIKeyRequestVerify
	proof   *tokens.APIKeyProofVerify
}

// NewAPIKeyVerifier creates a verifier of what the API keys of purpose signed, reading them through
// reader. A key made for another purpose verifies nothing, so NewAPIKeyVerifier refuses an empty
// one.
func NewAPIKeyVerifier(reader APIKeyReader, purpose APIKeyPurpose) (APIKeyVerifier, error) {
	keys, err := tokens.NewAPIKeys(newAPIKeyExportGrpc(reader), purpose)
	if err != nil {
		return nil, fmt.Errorf("(NewAPIKeyVerifier) new api keys: %w", err)
	}

	return &apiKeyVerifier{
		request: tokens.NewAPIKeyRequestVerify(keys),
		proof:   tokens.NewAPIKeyProofVerify(keys),
	}, nil
}

func (verifier *apiKeyVerifier) VerifyRequestToken(
	ctx context.Context, req *VerifyRequestTokenRequest,
) (*APIKeyRequest, error) {
	return verifier.request.Exec(ctx, req)
}

func (verifier *apiKeyVerifier) VerifyProof(ctx context.Context, req *VerifyProofRequest) (*APIKeyProof, error) {
	return verifier.proof.Exec(ctx, req)
}
