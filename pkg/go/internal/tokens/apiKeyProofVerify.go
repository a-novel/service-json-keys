package tokens

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2/jwa"
)

// TypProof is the media type the header of a proof declares.
const TypProof jwa.Typ = "proof+jwt"

// APIKeyProofVerifyRequest holds the parameters for an [APIKeyProofVerify.Exec] call.
type APIKeyProofVerifyRequest struct {
	// Proof is the proof to verify, a compact JWT.
	Proof string
	// Claims receives the claims of the proof, which say what the proof vouches for. It is a
	// non-nil pointer, as the destination of json.Unmarshal is.
	Claims any
}

// APIKeyProof is a proof an API key signed.
type APIKeyProof struct {
	// KeyID is the ID of the API key that signed the proof.
	KeyID string
}

// An APIKeyProofVerify verifies a proof: a JWT of type "proof+jwt" the owner of an API key signs for
// one entry it creates. A proof never expires, and one signed by a key deactivated since still
// verifies.
type APIKeyProofVerify struct {
	keys  *APIKeys
	token apiKeyToken
}

func NewAPIKeyProofVerify(keys *APIKeys) *APIKeyProofVerify {
	return &APIKeyProofVerify{keys: keys, token: keys.newToken(TypProof, true, 0)}
}

// Exec verifies the proof of the request and decodes its claims. A proof the key does not vouch for
// fails with an error matching [jwt.ErrTokenRejected]. The caller compares the claims with the entry
// it holds.
func (service *APIKeyProofVerify) Exec(
	ctx context.Context, request *APIKeyProofVerifyRequest,
) (*APIKeyProof, error) {
	ctx, span := otel.Tracer().Start(ctx, "tokens.APIKeyProofVerify")
	defer span.End()

	key, err := service.keys.consume(ctx, service.token, request.Proof, request.Claims)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.String("key.id", key.ID))

	return &APIKeyProof{KeyID: key.ID}, nil
}
