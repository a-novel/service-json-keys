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

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
)

func TestGrpcJwkGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *core.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *protogen.JwkGetRequest

		serviceMock *serviceMock

		expect       *protogen.JwkGetResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &protogen.JwkGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				resp: &core.Jwk{
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
			name: "Error/InvalidID",

			request: &protogen.JwkGetRequest{
				Id: "not-a-uuid",
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/NotFound",

			request: &protogen.JwkGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				err: core.ErrJwkNotFound,
			},

			expectStatus: codes.NotFound,
		},
		{
			name: "Error/Internal",

			request: &protogen.JwkGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				err: errFoo,
			},

			expectStatus: codes.Internal,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockGrpcJwkGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.JwkSelectRequest{
						ID: uuid.MustParse(testCase.request.GetId()),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewGrpcJwkGet(service)

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
