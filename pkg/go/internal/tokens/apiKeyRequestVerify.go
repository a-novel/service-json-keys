package tokens

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
)

// APIKeyRequestLifetime is the longest a request token may live.
const APIKeyRequestLifetime = 5 * time.Minute

// ErrRequestBody is returned when the body of a request fails to read.
var ErrRequestBody = errors.New("read request body")

// APIKeyRequestVerifyRequest holds the parameters for an [APIKeyRequestVerify.Exec] call.
type APIKeyRequestVerifyRequest struct {
	// Token is the request token of the request, a compact JWT.
	Token string
	// Body is the body of the request. It is read once the token passes every other check, so a
	// caller without a key never makes the server read one.
	Body io.Reader
}

// APIKeyRequest is a request an API key authenticated.
type APIKeyRequest struct {
	// KeyID is the ID of the API key that signed the request token.
	KeyID string
	// Body is the body the request token was signed for.
	Body []byte
}

// apiKeyRequestClaims are the claims a request token adds to the registered ones.
type apiKeyRequestClaims struct {
	// BodySHA256 is the SHA-256 of the body of the request, in lowercase hexadecimal.
	BodySHA256 string `json:"bodySHA256"` //nolint:tagliatelle // The token names the claim.
}

// An APIKeyRequestVerify authenticates a request by its request token: a JWT of type "JWT" the owner
// of an API key signs for one request, which lives [APIKeyRequestLifetime] at most and carries the
// SHA-256 of the body. Only a key that is still active authenticates.
//
// A request token can be replayed until it expires, with the body it was signed for alone. The
// verifier keeps no token, so the request a key authenticates must be safe to repeat.
type APIKeyRequestVerify struct {
	keys  *APIKeys
	token apiKeyToken
}

func NewAPIKeyRequestVerify(keys *APIKeys) *APIKeyRequestVerify {
	return &APIKeyRequestVerify{keys: keys, token: keys.newToken(jwa.TypJWT, false, APIKeyRequestLifetime)}
}

// Exec verifies the token of the request, then reads its body and compares it with the digest the
// token carries. A token or a body the key does not vouch for fails with an error matching
// [jwt.ErrTokenRejected], and a body that fails to read with one matching [ErrRequestBody].
func (service *APIKeyRequestVerify) Exec(
	ctx context.Context, request *APIKeyRequestVerifyRequest,
) (*APIKeyRequest, error) {
	ctx, span := otel.Tracer().Start(ctx, "tokens.APIKeyRequestVerify")
	defer span.End()

	var claims apiKeyRequestClaims

	key, err := service.keys.consume(ctx, service.token, request.Token, &claims)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.String("key.id", key.ID))

	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrRequestBody, err))
	}

	digest := sha256.Sum256(body)
	if claims.BodySHA256 != hex.EncodeToString(digest[:]) {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: the body is not the one the request token was signed for", jwt.ErrTokenRejected,
		))
	}

	return &APIKeyRequest{KeyID: key.ID, Body: body}, nil
}
