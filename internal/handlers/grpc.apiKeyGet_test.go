package handlers_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

func TestGrpcAPIKeyGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *core.APIKey
		err  error
	}

	testCases := []struct {
		name string

		request *jsonkeysv2.APIKeyGetRequest

		serviceMock *serviceMock

		expect       *jsonkeysv2.APIKeyGetResponse
		expectStatus codes.Code
		// expectCause is the error a server fault keeps, so the call's server span names its type.
		expectCause error
	}{
		{
			name: "Success",

			request: &jsonkeysv2.APIKeyGetRequest{Id: "test-id"},

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
					Leaked:      true,
				},
			},

			expectStatus: codes.OK,
			expect: &jsonkeysv2.APIKeyGetResponse{
				ApiKey: &jsonkeysv2.APIKey{
					Id:      "test-id",
					Purpose: "test-purpose",
					Jwk: &jsonkeysv2.Jwk{
						Kty:     "test-kty",
						Use:     "test-use",
						KeyOps:  []string{"verify"},
						Alg:     "test-alg",
						Kid:     "test-id",
						Payload: []byte(`{"x":"test-x"}`),
					},
					Deactivated: true,
					Leaked:      true,
				},
			},
		},
		{
			name: "Error/NotFound",

			request: &jsonkeysv2.APIKeyGetRequest{Id: "test-id"},

			serviceMock: &serviceMock{err: core.ErrAPIKeyNotFound},

			expectStatus: codes.NotFound,
		},
		{
			name: "Error/Internal",

			request: &jsonkeysv2.APIKeyGetRequest{Id: "test-id"},

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: codes.Internal,
			expectCause:  errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockGrpcAPIKeyGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.APIKeySelectRequest{ID: testCase.request.GetId()}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewGrpcAPIKeyGet(service)

			res, err := handler.APIKeyGet(t.Context(), testCase.request)
			resSt, ok := status.FromError(err)
			require.True(t, ok, resSt.Code().String())
			require.Equal(
				t,
				testCase.expectStatus, resSt.Code(),
				"expected status code %s, got %s (%v)", testCase.expectStatus, resSt.Code(), err,
			)
			require.Equal(t, testCase.expect, res)

			if testCase.expectCause != nil {
				require.ErrorIs(t, err, testCase.expectCause)
			}
		})
	}
}
