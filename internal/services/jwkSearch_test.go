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

func TestJwkSearch(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type repositorySearchMock struct {
		resp []*dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *services.Jwk
		err  error
	}

	testCases := []struct {
		name string

		request *services.JwkSearchRequest

		repositorySearchMock *repositorySearchMock
		serviceExtractMock   []*serviceExtractMock

		expect    []*services.Jwk
		expectErr error
	}{
		{
			name: "Success",

			request: &services.JwkSearchRequest{
				Usage:   "test-usage",
				Private: true,
			},

			repositorySearchMock: &repositorySearchMock{
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
				{
					resp: &services.Jwk{
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

			expect: []*services.Jwk{
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
			name: "Error/Extract",

			request: &services.JwkSearchRequest{
				Usage:   "test-usage",
				Private: true,
			},

			repositorySearchMock: &repositorySearchMock{
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
				{
					err: errFoo,
				},
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Search",

			request: &services.JwkSearchRequest{
				Usage:   "test-usage",
				Private: true,
			},

			repositorySearchMock: &repositorySearchMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repositorySearch := servicesmocks.NewMockJwkSearchRepository(t)
			serviceExtract := servicesmocks.NewMockJwkSearchServiceExtract(t)

			if testCase.repositorySearchMock != nil {
				repositorySearch.EXPECT().
					Exec(mock.Anything, &dao.JwkSearchRequest{
						Usage: testCase.request.Usage,
					}).
					Return(testCase.repositorySearchMock.resp, testCase.repositorySearchMock.err)
			}

			if testCase.serviceExtractMock != nil {
				require.GreaterOrEqual(t, len(testCase.serviceExtractMock), len(testCase.repositorySearchMock.resp))

				for i, extracted := range testCase.serviceExtractMock {
					serviceExtract.EXPECT().
						Exec(mock.Anything, &services.JwkExtractRequest{
							Jwk:     testCase.repositorySearchMock.resp[i],
							Private: testCase.request.Private,
						}).
						Return(extracted.resp, extracted.err).
						Once()
				}
			}

			service := services.NewJwkSearch(repositorySearch, serviceExtract)

			res, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			repositorySearch.AssertExpectations(t)
			serviceExtract.AssertExpectations(t)
		})
	}
}
