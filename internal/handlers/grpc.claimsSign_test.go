package handlers_test

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/grpcf"

	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-json-keys/v2/internal/handlers/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func TestGrpcClaimsSign(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  any
		resp string
		err  error
	}

	testCases := []struct {
		name string

		request *protogen.ClaimsSignRequest

		serviceMock *serviceMock

		expect       *protogen.ClaimsSignResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &protogen.ClaimsSignRequest{
				Payload: lo.Must(grpcf.InterfaceToProtoAny(map[string]any{"message": "hello world"})),
				Usage:   "test-usage",
			},

			serviceMock: &serviceMock{
				req:  map[string]any{"message": "hello world"},
				resp: "access-token",
			},

			expectStatus: codes.OK,
			expect: &protogen.ClaimsSignResponse{
				Token: "access-token",
			},
		},
		{
			name: "Error/BadConfig",

			request: &protogen.ClaimsSignRequest{
				Payload: lo.Must(grpcf.InterfaceToProtoAny(map[string]any{"message": "hello world"})),
				Usage:   "test-usage",
			},

			serviceMock: &serviceMock{
				req: map[string]any{"message": "hello world"},
				err: services.ErrConfigNotFound,
			},

			expectStatus: codes.Unavailable,
		},
		{
			name: "Error/Internal",

			request: &protogen.ClaimsSignRequest{
				Payload: lo.Must(grpcf.InterfaceToProtoAny(map[string]any{"message": "hello world"})),
				Usage:   "test-usage",
			},

			serviceMock: &serviceMock{
				req: map[string]any{"message": "hello world"},
				err: errFoo,
			},

			expectStatus: codes.Internal,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockGrpcClaimsSignService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &services.ClaimsSignRequest{
						Claims: testCase.serviceMock.req,
						Usage:  testCase.request.GetUsage(),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewGrpcClaimsSign(service)

			res, err := handler.ClaimsSign(t.Context(), testCase.request)
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
