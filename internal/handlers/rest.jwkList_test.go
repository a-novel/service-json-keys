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

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func TestRestJwkList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp []*services.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/jwks?usage=test-usage", nil),

			serviceMock: &serviceMock{
				resp: []*services.Jwk{
					{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: jwa.KeyOps{jwa.KeyOpSign, jwa.KeyOpVerify},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000001",
						},
						Payload: json.RawMessage(`{"x":"test-x"}`),
					},
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: []any{
				map[string]any{
					"kty":     "test-kty",
					"use":     "test-use",
					"key_ops": []any{"sign", "verify"},
					"alg":     "test-alg",
					"kid":     "00000000-0000-0000-0000-000000000001",
					"x":       "test-x",
				},
			},
		},
		{
			name: "Success/Empty",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/jwks?usage=test-usage", nil),

			serviceMock: &serviceMock{
				resp: []*services.Jwk{},
			},

			expectStatus: http.StatusOK,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/jwks?usage=test-usage", nil),

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
					Exec(mock.Anything, &services.JwkSearchRequest{
						Usage: testCase.request.URL.Query().Get("usage"),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestJwkList(service, config.LoggerDev)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, testCase.request)

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

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
