package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/internal/api"
	"github.com/a-novel/service-json-keys/internal/api/codegen"
	apimocks "github.com/a-novel/service-json-keys/internal/api/mocks"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
)

func TestSignClaims(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type signClaimsData struct {
		res string
		err error
	}

	testCases := []struct {
		name string

		req    codegen.SignClaimsReq
		params codegen.SignClaimsParams

		signClaimsData *signClaimsData

		expect    codegen.SignClaimsRes
		expectErr error
	}{
		{
			name: "Success",

			req: codegen.SignClaimsReq{
				"test-claim":    []byte(`"test-value"`),
				"another-claim": []byte("12345"),
			},

			params: codegen.SignClaimsParams{
				Usage: "test-usage",
			},

			signClaimsData: &signClaimsData{
				res: "test-token",
			},

			expect: &codegen.Token{Token: "test-token"},
		},
		{
			name: "Error",

			req: codegen.SignClaimsReq{
				"test-claim":    []byte(`"test-value"`),
				"another-claim": []byte("12345"),
			},

			params: codegen.SignClaimsParams{
				Usage: "test-usage",
			},

			signClaimsData: &signClaimsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockSignClaimsService(t)

			if testCase.signClaimsData != nil {
				source.EXPECT().
					SignClaims(mock.Anything, services.SignClaimsRequest{
						Claims: testCase.req,
						Usage:  models.KeyUsage(testCase.params.Usage),
					}).
					Return(testCase.signClaimsData.res, testCase.signClaimsData.err)
			}

			handler := api.API{SignClaimsService: source}

			res, err := handler.SignClaims(t.Context(), testCase.req, testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
