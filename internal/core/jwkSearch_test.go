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

func TestJwkSearch(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type daoSearchMock struct {
		resp []*dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *core.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *core.JwkSearchRequest

		daoSearchMock      *daoSearchMock
		serviceExtractMock []*serviceExtractMock

		expect    []*core.Jwk
		expectErr error
	}{
		{
			name: "Success",

			request: &core.JwkSearchRequest{
				Usage:   "test-usage",
				Private: true,
			},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMg",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			serviceExtractMock: []*serviceExtractMock{
				{
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
				{
					resp: &core.Jwk{
						JWKCommon: jwa.JWKCommon{
							KTY: "test-kty",
							Use: "test-use",
							Alg: "test-alg",
							KID: "00000000-0000-0000-0000-000000000001",
						},
						Payload: json.RawMessage(`{"message":"hello world"}`),
					},
				},
			},

			expect: []*core.Jwk{
				{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: "test-alg",
						KID: "00000000-0000-0000-0000-000000000002",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
				{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: "test-alg",
						KID: "00000000-0000-0000-0000-000000000001",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},
		},
		{
			name: "Success/Empty",

			request: &core.JwkSearchRequest{
				Usage:   "test-usage",
				Private: true,
			},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{},
			},

			expect: []*core.Jwk{},
		},
		{
			name: "Error/Extract",

			request: &core.JwkSearchRequest{
				Usage:   "test-usage",
				Private: true,
			},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMg",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			serviceExtractMock: []*serviceExtractMock{
				{
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
				{
					err: errFoo,
				},
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Search",

			request: &core.JwkSearchRequest{
				Usage:   "test-usage",
				Private: true,
			},

			daoSearchMock: &daoSearchMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSearch := coremocks.NewMockJwkSearchDao(t)
			serviceExtract := coremocks.NewMockJwkSearchServiceExtract(t)

			if testCase.daoSearchMock != nil {
				daoSearch.EXPECT().
					Exec(mock.Anything, &dao.JwkSearchRequest{
						Usage: testCase.request.Usage,
					}).
					Return(testCase.daoSearchMock.resp, testCase.daoSearchMock.err)
			}

			if testCase.serviceExtractMock != nil {
				require.GreaterOrEqual(t, len(testCase.serviceExtractMock), len(testCase.daoSearchMock.resp))

				for i, extracted := range testCase.serviceExtractMock {
					serviceExtract.EXPECT().
						Exec(mock.Anything, &core.JwkExtractRequest{
							Jwk:     testCase.daoSearchMock.resp[i],
							Private: testCase.request.Private,
						}).
						Return(extracted.resp, extracted.err).
						Once()
				}
			}

			service := core.NewJwkSearch(daoSearch, serviceExtract)

			res, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			daoSearch.AssertExpectations(t)
			serviceExtract.AssertExpectations(t)
		})
	}
}
