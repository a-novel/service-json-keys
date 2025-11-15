package handlers_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func TestJwkGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceJwkGetMock struct {
		resp *services.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *protogen.JwkGetRequest

		serviceJwkGetMock *serviceJwkGetMock

		expect       *protogen.JwkGetResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &protogen.JwkGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceJwkGetMock: &serviceJwkGetMock{
				resp: &services.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: jwa.KeyOps{jwa.KeyOpSign, jwa.KeyOpVerify},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000001",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expectStatus: codes.OK,
			expect: &protogen.JwkGetResponse{
				Jwk: &protogen.Jwk{
					Kty:     "test-kty",
					Use:     "test-use",
					KeyOps:  []string{"sign", "verify"},
					Alg:     "test-alg",
					Kid:     "00000000-0000-0000-0000-000000000001",
					Payload: []byte(`{"message":"hello world"}`),
				},
			},
		},
		{
			name: "Error/NotFound",

			request: &protogen.JwkGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceJwkGetMock: &serviceJwkGetMock{
				err: dao.ErrJwkSelectNotFound,
			},

			expectStatus: codes.NotFound,
		},
		{
			name: "Error/Internal",

			request: &protogen.JwkGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceJwkGetMock: &serviceJwkGetMock{
				err: errFoo,
			},

			expectStatus: codes.Internal,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockJwkGetService(t)

			if testCase.serviceJwkGetMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &services.JwkSelectRequest{
						ID: uuid.MustParse(testCase.request.GetId()),
					}).
					Return(testCase.serviceJwkGetMock.resp, testCase.serviceJwkGetMock.err)
			}

			handler := handlers.NewJwkGet(service)

			res, err := handler.JwkGet(t.Context(), testCase.request)
			resSt, ok := status.FromError(err)
			require.True(t, ok, resSt.Code().String())
			require.Equal(
				t,
				testCase.expectStatus, resSt.Code(),
				"expected status code %s, got %s (%v)", testCase.expectStatus, resSt.Code(), err,
			)
			require.Equal(t, testCase.expect, res)

			service.AssertExpectations(t)
		})
	}
}
