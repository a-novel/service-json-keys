package handlers_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
	protogen "github.com/a-novel/service-json-keys/v2/internal/handlers/proto/gen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func TestJwkList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceJwkSearchMock struct {
		resp []*services.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *protogen.JwkListRequest

		serviceJwkSearchMock *serviceJwkSearchMock

		expect       *protogen.JwkListResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &protogen.JwkListRequest{
				Usage:   "test-usage",
				Private: true,
			},

			serviceJwkSearchMock: &serviceJwkSearchMock{
				resp: []*services.Jwk{
					{
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
			},

			expectStatus: codes.OK,
			expect: &protogen.JwkListResponse{
				Keys: []*protogen.Jwk{
					{
						Kty:     "test-kty",
						Use:     "test-use",
						KeyOps:  []string{"sign", "verify"},
						Alg:     "test-alg",
						Kid:     "00000000-0000-0000-0000-000000000001",
						Payload: []byte(`{"message":"hello world"}`),
					},
				},
			},
		},
		{
			name: "Error/Internal",

			request: &protogen.JwkListRequest{
				Usage:   "test-usage",
				Private: true,
			},

			serviceJwkSearchMock: &serviceJwkSearchMock{
				err: errFoo,
			},

			expectStatus: codes.Internal,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockJwkListService(t)

			if testCase.serviceJwkSearchMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &services.JwkSearchRequest{
						Usage:   testCase.request.GetUsage(),
						Private: testCase.request.GetPrivate(),
					}).
					Return(testCase.serviceJwkSearchMock.resp, testCase.serviceJwkSearchMock.err)
			}

			handler := handlers.NewJwkList(service)

			res, err := handler.JwkList(t.Context(), testCase.request)
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
