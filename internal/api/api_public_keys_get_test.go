package api_test

import (
	"errors"
	"testing"

	"github.com/go-faster/jx"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/api"
	apimocks "github.com/a-novel/service-json-keys/internal/api/mocks"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models/api"
)

func TestGetPublicKey(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type selectKeyData struct {
		res *jwa.JWK
		err error
	}

	testCases := []struct {
		name string

		params apimodels.GetPublicKeyParams

		selectKeyData *selectKeyData

		expect    apimodels.GetPublicKeyRes
		expectErr error
	}{
		{
			name: "Success",

			params: apimodels.GetPublicKeyParams{
				Kid: apimodels.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			selectKeyData: &selectKeyData{
				res: &jwa.JWK{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-keyops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000001",
					},
					Payload: []byte(`{"test":"payload"}`),
				},
			},

			expect: &apimodels.JWK{
				Kty:    "test-kty",
				Use:    "test-use",
				KeyOps: []apimodels.KeyOp{"test-keyops"},
				Alg:    "test-alg",
				Kid: apimodels.OptKID{
					Value: apimodels.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Set:   true,
				},
				AdditionalProps: apimodels.JWKAdditional{
					"test": jx.Raw(`"payload"`),
				},
			},
		},
		{
			name: "KeyNotFound",

			params: apimodels.GetPublicKeyParams{
				Kid: apimodels.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			selectKeyData: &selectKeyData{
				err: dao.ErrKeyNotFound,
			},

			expect: &apimodels.NotFoundError{Error: "key not found"},
		},
		{
			name: "Error",

			params: apimodels.GetPublicKeyParams{
				Kid: apimodels.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			selectKeyData: &selectKeyData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockSelectKeyService(t)

			if testCase.selectKeyData != nil {
				source.EXPECT().
					SelectKey(mock.Anything, services.SelectKeyRequest{
						ID:      uuid.UUID(testCase.params.Kid),
						Private: false,
					}).
					Return(testCase.selectKeyData.res, testCase.selectKeyData.err)
			}

			handler := api.API{SelectKeyService: source}

			res, err := handler.GetPublicKey(t.Context(), testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
