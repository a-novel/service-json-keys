package servicejsonkeys_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"
)

// unavailableAPIKeys is a reader of API keys whose service is down.
type unavailableAPIKeys struct{}

func (unavailableAPIKeys) APIKeyGet(
	context.Context, *servicejsonkeys.APIKeyGetRequest, ...grpc.CallOption,
) (*servicejsonkeys.APIKeyGetResponse, error) {
	return nil, status.Error(codes.Unavailable, "unavailable")
}

func TestNewAPIKeyGate(t *testing.T) {
	t.Parallel()

	const (
		purpose = servicejsonkeys.APIKeyPurposePublishNarrativeEnginePlugin
		body    = `{"plugin":"scenes-composer","version":"1.0.0"}`
		// keyHeader is where the test handler writes the ID of the key the gate authenticated.
		keyHeader = "X-Test-Key"
	)

	key := servicejsonkeystest.NewAPIKey(t, purpose)
	unknown := servicejsonkeystest.NewAPIKey(t, purpose)

	valid := "Bearer " + key.Sign(t, servicejsonkeystest.TypRequestToken, key.RequestClaims([]byte(body)))

	testCases := []struct {
		name string

		// reader serves the API keys, the key of the test when nil.
		reader        servicejsonkeys.APIKeyReader
		config        servicejsonkeys.APIKeyGateConfig
		authorization string
		maxBodySize   int64
		brokenBody    bool

		expectStatus int
		// expectKey is the ID of the key the handler finds in its context.
		expectKey string
	}{
		{
			name: "Success",

			authorization: valid,

			expectStatus: http.StatusOK,
			expectKey:    key.ID,
		},
		{
			name: "Success/SchemeCase",

			authorization: "bearer" + strings.TrimPrefix(valid, "Bearer"),

			expectStatus: http.StatusOK,
			expectKey:    key.ID,
		},
		{
			// A local stack whose callers hold no key lets them in, unauthenticated.
			name: "Success/TokenOptional",

			config: servicejsonkeys.APIKeyGateConfig{TokenOptional: true},

			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NoToken",

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/OtherScheme",

			authorization: "Basic " + strings.TrimPrefix(valid, "Bearer "),

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/RefusedToken",

			authorization: "Bearer " + unknown.Sign(
				t, servicejsonkeystest.TypRequestToken, unknown.RequestClaims([]byte(body)),
			),

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A token is optional, never ignored: one the gate refuses still closes it.
			name: "Error/TokenOptionalRefusedToken",

			config: servicejsonkeys.APIKeyGateConfig{TokenOptional: true},
			authorization: "Bearer " + unknown.Sign(
				t, servicejsonkeystest.TypRequestToken, unknown.RequestClaims([]byte(body)),
			),

			expectStatus: http.StatusUnauthorized,
		},
		{
			// The token vouches for another body than the one the request carries.
			name: "Error/OtherBody",

			authorization: "Bearer " + key.Sign(
				t, servicejsonkeystest.TypRequestToken, key.RequestClaims([]byte(`{"plugin":"other"}`)),
			),

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/BodyTooLarge",

			authorization: valid,
			maxBodySize:   int64(len(body) - 1),

			expectStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "Error/UnreadableBody",

			authorization: valid,
			brokenBody:    true,

			expectStatus: http.StatusBadRequest,
		},
		{
			// A key the service could not serve is the server's fault, never the caller's.
			name: "Error/KeyUnavailable",

			reader:        unavailableAPIKeys{},
			authorization: valid,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reader := testCase.reader
			if reader == nil {
				reader = servicejsonkeystest.APIKeys{key}
			}

			verifier, err := servicejsonkeys.NewAPIKeyVerifier(reader, purpose)
			require.NoError(t, err)

			// The handler reads the body the gate checked, and the key that authenticated the request.
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received, err := io.ReadAll(r.Body)
				if err != nil || string(received) != body {
					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				keyID, _ := servicejsonkeys.RequestAPIKey(r.Context())
				w.Header().Set(keyHeader, keyID)
				w.WriteHeader(http.StatusOK)
			})

			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPost, "/v1/plugin/version", strings.NewReader(body),
			)
			if testCase.authorization != "" {
				request.Header.Set("Authorization", testCase.authorization)
			}

			recorder := httptest.NewRecorder()
			if testCase.maxBodySize > 0 {
				request.Body = http.MaxBytesReader(recorder, request.Body, testCase.maxBodySize)
			}

			if testCase.brokenBody {
				request.Body = io.NopCloser(iotest.ErrReader(errors.New("foo")))
			}

			gate := servicejsonkeys.NewAPIKeyGate(verifier, testCase.config, configtest.Logger)
			gate(next).ServeHTTP(recorder, request)

			require.Equal(t, testCase.expectStatus, recorder.Code)
			require.Equal(t, testCase.expectKey, recorder.Header().Get(keyHeader))

			// Every refused token reads alike, so the answer never says which check failed.
			if testCase.expectStatus == http.StatusUnauthorized {
				require.Equal(t, "Bearer", recorder.Header().Get("WWW-Authenticate"))
				require.Equal(t, http.StatusText(http.StatusUnauthorized)+"\n", recorder.Body.String())
			}
		})
	}
}
