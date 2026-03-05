package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func TestJwkGetPublic(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceJwkGetMock struct {
		resp *services.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request

		serviceJwkGetMock *serviceJwkGetMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequest(http.MethodGet, "/jwk?id=00000000-0000-0000-0000-000000000001", nil),

			serviceJwkGetMock: &serviceJwkGetMock{
				resp: &services.Jwk{
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

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"kty":     "test-kty",
				"use":     "test-use",
				"key_ops": []any{"sign", "verify"},
				"alg":     "test-alg",
				"kid":     "00000000-0000-0000-0000-000000000001",
				"x":       "test-x",
			},
		},
		{
			name: "Error/InvalidID",

			request: httptest.NewRequest(http.MethodGet, "/jwk?id=not-a-uuid", nil),

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: httptest.NewRequest(http.MethodGet, "/jwk?id=00000000-0000-0000-0000-000000000001", nil),

			serviceJwkGetMock: &serviceJwkGetMock{
				err: dao.ErrJwkSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequest(http.MethodGet, "/jwk?id=00000000-0000-0000-0000-000000000001", nil),

			serviceJwkGetMock: &serviceJwkGetMock{
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockJwkGetPublicService(t)

			if testCase.serviceJwkGetMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &services.JwkSelectRequest{
						ID: uuid.MustParse(testCase.request.URL.Query().Get("id")),
					}).
					Return(testCase.serviceJwkGetMock.resp, testCase.serviceJwkGetMock.err)
			}

			handler := handlers.NewJwkGetPublic(service)
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
