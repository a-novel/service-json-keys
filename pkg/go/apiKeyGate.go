package servicejsonkeys

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"

	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
)

// apiKeyGateErrors answers 401 to a request the gate refuses, and 400 to a body that failed to
// read, as a handler's own decoding does. A body past the route's limit answers 413, and a key the
// service could not serve 500.
var apiKeyGateErrors = httpf.ErrMap{
	ErrTokenRejected:      http.StatusUnauthorized,
	tokens.ErrRequestBody: http.StatusBadRequest,
}

// errRequestTokenMissing refuses a request whose Authorization header carries no Bearer token.
var errRequestTokenMissing = fmt.Errorf("%w: expected a request token as a Bearer token", ErrTokenRejected)

// APIKeyGateConfig configures [NewAPIKeyGate].
type APIKeyGateConfig struct {
	// TokenOptional admits a request that carries no Authorization header, unauthenticated. Only a
	// local or test stack sets it, whose callers hold no key: production leaves it unset.
	TokenOptional bool
}

// apiKeyContextKey is the context key of the ID of the API key that authenticated a request.
type apiKeyContextKey struct{}

// NewAPIKeyGate returns the middleware that refuses a request unless verifier authenticates it,
// and logs each refusal through logger. The owner of an API key sends the request token as a Bearer
// token in the Authorization header (RFC 6750 §2.1). The handlers read the body the token was
// checked against, and the ID of the key through [RequestAPIKey].
//
// The gate fails closed: it refuses a request without a token, unless config says otherwise. It
// holds a request's body in memory, so mount it under a body limit, as every route of httpf.NewMux
// is.
func NewAPIKeyGate(
	verifier APIKeyVerifier, config APIKeyGateConfig, logger logging.Log,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if authorization == "" && config.TokenOptional {
				next.ServeHTTP(w, r)

				return
			}

			// RFC 9110 §11.1: the name of an authentication scheme ignores case.
			scheme, token, _ := strings.Cut(authorization, " ")
			if !strings.EqualFold(scheme, "Bearer") {
				refuseAPIKey(w, r, logger, errRequestTokenMissing)

				return
			}

			// RFC 6750 §2.1: one or more spaces separate the scheme from the token.
			token = strings.TrimLeft(token, " ")

			request, err := verifier.VerifyRequestToken(r.Context(), &VerifyRequestTokenRequest{Token: token, Body: r.Body})
			if err != nil {
				refuseAPIKey(w, r, logger, err)

				return
			}

			// A handler changes nothing of the request it is handed, so the body goes on a copy.
			r = r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, request.KeyID))
			r.Body = io.NopCloser(bytes.NewReader(request.Body))

			next.ServeHTTP(w, r)
		})
	}
}

// RequestAPIKey returns the ID of the API key that authenticated the request of ctx. It reports
// false for a request the gate of [NewAPIKeyGate] admitted without a token, or never saw.
func RequestAPIKey(ctx context.Context) (string, bool) {
	keyID, ok := ctx.Value(apiKeyContextKey{}).(string)

	return keyID, ok
}

// refuseAPIKey answers a request the gate refuses, through [httpf.HandleError], which logs err.
//
// A refused token answers 401 with the WWW-Authenticate challenge RFC 9110 §15.5.2 requires. The
// challenge carries no error code, which RFC 6750 §3 only recommends: every refusal reads alike, so
// the answer never says which check failed. The log does.
func refuseAPIKey(w http.ResponseWriter, r *http.Request, logger logging.Log, err error) {
	if errors.Is(err, ErrTokenRejected) {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}

	httpf.HandleError(r.Context(), logger, w, trace.SpanFromContext(r.Context()), apiKeyGateErrors, err)
}
