package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
)

func TestRestJwkList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp []*core.Jwk
		err  error
	}

	key := &core.Jwk{
		JWKCommon: jwa.JWKCommon{
			KTY:    "test-kty",
			Use:    "test-use",
			KeyOps: jwa.KeyOps{jwa.KeyOpSign, jwa.KeyOpVerify},
			Alg:    "test-alg",
			KID:    "00000000-0000-0000-0000-000000000001",
		},
		Payload: json.RawMessage(`{"x":"test-x"}`),
	}

	keyAnswer := map[string]any{
		"kty":     "test-kty",
		"use":     "test-use",
		"key_ops": []any{"sign", "verify"},
		"alg":     "test-alg",
		"kid":     "00000000-0000-0000-0000-000000000001",
		"x":       "test-x",
	}

	// The SHA-256 of the JSON answers holding key alone, and no key.
	keyETag := `"ciIFXG359yAXz4WSHLz3UGEu-BkVwVSIHZosYj-pFPA"`
	emptyETag := `"T1PNoYwrqgwDVLtfmj7L5e0Sq02OEbqHPC8RFhICuUU"`

	testCases := []struct {
		name string

		request *http.Request
		// ifNoneMatch, when set, revalidates a stored answer as a cache does.
		ifNoneMatch string

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
		// expectETag is the validator of a successful answer, a 304 included.
		expectETag string
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/jwks?usage=test-usage", nil),

			serviceMock: &serviceMock{
				resp: []*core.Jwk{key},
			},

			expectStatus:   http.StatusOK,
			expectResponse: []any{keyAnswer},
			expectETag:     keyETag,
		},
		{
			name: "Success/NotModified",

			request:     httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/jwks?usage=test-usage", nil),
			ifNoneMatch: keyETag,

			serviceMock: &serviceMock{
				resp: []*core.Jwk{key},
			},

			expectStatus: http.StatusNotModified,
			expectETag:   keyETag,
		},
		{
			// A rotated key changes the set, so the stored answer no longer validates.
			name: "Success/Rotated",

			request:     httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/jwks?usage=test-usage", nil),
			ifNoneMatch: emptyETag,

			serviceMock: &serviceMock{
				resp: []*core.Jwk{key},
			},

			expectStatus:   http.StatusOK,
			expectResponse: []any{keyAnswer},
			expectETag:     keyETag,
		},
		{
			name: "Success/Empty",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/jwks?usage=test-usage", nil),

			serviceMock: &serviceMock{
				resp: []*core.Jwk{},
			},

			expectStatus:   http.StatusOK,
			expectResponse: []any{},
			expectETag:     emptyETag,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/jwks?usage=test-usage", nil),

			serviceMock: &serviceMock{
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestJwkListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.JwkSearchRequest{
						Usage: testCase.request.URL.Query().Get("usage"),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			if testCase.ifNoneMatch != "" {
				testCase.request.Header.Set("If-None-Match", testCase.ifNoneMatch)
			}

			handler := handlers.NewRestJwkList(service, config.LoggerDev)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, testCase.request)

			res := w.Result()

			service.AssertExpectations(t)
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectETag != "" {
				require.Equal(t, testCase.expectETag, res.Header.Get("ETag"))
				require.Equal(t, "no-cache", res.Header.Get("Cache-Control"))
			}

			if testCase.expectStatus == http.StatusNotModified {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))
				require.Empty(t, data)
			}

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
