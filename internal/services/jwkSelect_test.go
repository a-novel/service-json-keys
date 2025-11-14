package services_test

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

	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	testutils "github.com/a-novel/service-json-keys/internal/lib/test"
	"github.com/a-novel/service-json-keys/internal/services"
	servicesmocks "github.com/a-novel/service-json-keys/internal/services/mocks"
)

func TestJwkSelect(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type repositorySelectMock struct {
		resp *dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *services.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *services.JwkSelectRequest

		repositorySelectMock *repositorySelectMock
		serviceExtractMock   *serviceExtractMock

		expect    *services.Jwk
		expectErr error
	}{
		{
			name: "Success",

			request: &services.JwkSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Private: true,
			},

			repositorySelectMock: &repositorySelectMock{
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
				resp: &services.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: "test-alg",
						KID: "00000000-0000-0000-0000-000000000002",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &services.Jwk{
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

			request: &services.JwkSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Private: true,
			},

			repositorySelectMock: &repositorySelectMock{
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
			name: "Error/Select",

			request: &services.JwkSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Private: true,
			},

			repositorySelectMock: &repositorySelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repositorySelect := servicesmocks.NewMockJwkSelectRepository(t)
			serviceExtract := servicesmocks.NewMockJwkSelectServiceExtract(t)

			if testCase.repositorySelectMock != nil {
				repositorySelect.EXPECT().
					Exec(mock.Anything, &dao.JwkSelectRequest{
						ID: testCase.request.ID,
					}).
					Return(testCase.repositorySelectMock.resp, testCase.repositorySelectMock.err)
			}

			if testCase.serviceExtractMock != nil {
				serviceExtract.EXPECT().
					Exec(mock.Anything, &services.JwkExtractRequest{
						Jwk:     testCase.repositorySelectMock.resp,
						Private: testCase.request.Private,
					}).
					Return(testCase.serviceExtractMock.resp, testCase.serviceExtractMock.err).
					Once()
			}

			service := services.NewJwkSelect(repositorySelect, serviceExtract)

			res, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			repositorySelect.AssertExpectations(t)
			serviceExtract.AssertExpectations(t)
		})
	}
}
