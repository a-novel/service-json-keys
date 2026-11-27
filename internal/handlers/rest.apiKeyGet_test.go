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

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
)

func TestRestAPIKeyGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *core.APIKey
		err  error
	}

	testCases := []struct {
		name string

		id string

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			id: "test-id",

			serviceMock: &serviceMock{
				resp: &core.APIKey{
					ID:      "test-id",
					Purpose: "test-purpose",
					Key: &core.Jwk{
						KTY:     "test-kty",
						Use:     "test-use",
						KeyOps:  jwa.KeyOps{jwa.KeyOpVerify},
						Alg:     "test-alg",
						KID:     "test-id",
						Payload: json.RawMessage(`{"x":"test-x"}`),
					},
					Deactivated: true,
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"id":      "test-id",
				"purpose": "test-purpose",
				"key": map[string]any{
					"kty":     "test-kty",
					"use":     "test-use",
					"key_ops": []any{"verify"},
					"alg":     "test-alg",
					"kid":     "test-id",
					"x":       "test-x",
				},
				"deactivated": true,
				"leaked":      false,
			},
		},
		{
			name: "Error/NotFound",

			id: "test-id",

			serviceMock: &serviceMock{err: core.ErrAPIKeyNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			id: "test-id",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestAPIKeyGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.APIKeySelectRequest{ID: testCase.id}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/api-key?id="+testCase.id, nil)

			handler := handlers.NewRestAPIKeyGet(service, configtest.Logger)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, request)

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				// A cache revalidates the key on every use, so it learns a change of state at once.
				require.Equal(t, "no-cache", res.Header.Get("Cache-Control"))
				require.NotEmpty(t, res.Header.Get("ETag"))

				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
