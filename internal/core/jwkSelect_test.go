package core_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	testutils "github.com/a-novel/service-json-keys/v2/internal/lib/test"
)

func TestJwkSelect(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type daoSelectMock struct {
		resp *dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *core.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *core.JwkSelectRequest

		daoSelectMock      *daoSelectMock
		serviceExtractMock *serviceExtractMock

		expect    *core.Jwk
		expectErr error
	}{
		{
			name: "Success",

			request: &core.JwkSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Private: true,
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.Jwk{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      "test-usage",
					CreatedAt:  time.Now().Add(-time.Hour),
					ExpiresAt:  time.Now().Add(time.Hour),
				},
			},

			serviceExtractMock: &serviceExtractMock{
				resp: &core.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: "test-alg",
						KID: "00000000-0000-0000-0000-000000000002",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &core.Jwk{
				JWKCommon: jwa.JWKCommon{
					KTY: "test-kty",
					Use: "test-use",
					Alg: "test-alg",
					KID: "00000000-0000-0000-0000-000000000002",
				},
				Payload: json.RawMessage(`{"message":"hello world"}`),
			},
		},
		{
			name: "Error/Extract",

			request: &core.JwkSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Private: true,
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.Jwk{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      "test-usage",
					CreatedAt:  time.Now().Add(-time.Hour),
					ExpiresAt:  time.Now().Add(time.Hour),
				},
			},

			serviceExtractMock: &serviceExtractMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/NotFound",

			request: &core.JwkSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Private: true,
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrJwkSelectNotFound,
			},

			expectErr: core.ErrJwkNotFound,
		},
		{
			name: "Error/Select",

			request: &core.JwkSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Private: true,
			},

			daoSelectMock: &daoSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSelect := coremocks.NewMockJwkSelectDao(t)
			serviceExtract := coremocks.NewMockJwkSelectServiceExtract(t)

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.JwkSelectRequest{
						ID: testCase.request.ID,
					}).
					Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
			}

			if testCase.serviceExtractMock != nil {
				serviceExtract.EXPECT().
					Exec(mock.Anything, &core.JwkExtractRequest{
						Jwk:     testCase.daoSelectMock.resp,
						Private: testCase.request.Private,
					}).
					Return(testCase.serviceExtractMock.resp, testCase.serviceExtractMock.err).
					Once()
			}

			service := core.NewJwkSelect(daoSelect, serviceExtract)

			res, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			daoSelect.AssertExpectations(t)
			serviceExtract.AssertExpectations(t)
		})
	}
}
