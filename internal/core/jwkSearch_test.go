package core_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

func TestJwkSearch(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoSearchMock struct {
		resp []*dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *core.Jwk
		err  error
	}

	storedKeys := []*dao.Jwk{
		{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			PrivateKey: "cHJpdmF0ZS1rZXktMg",
			PublicKey:  new("cHVibGljLWtleS0y"),
			Usage:      "test-usage",
			CreatedAt:  time.Now().Add(-time.Hour),
			ExpiresAt:  time.Now().Add(time.Hour),
		},
		{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			PrivateKey: "cHJpdmF0ZS1rZXktMQ",
			PublicKey:  new("cHVibGljLWtleS0x"),
			Usage:      "test-usage",
			CreatedAt:  time.Now().Add(-time.Hour),
			ExpiresAt:  time.Now().Add(time.Hour),
		},
	}
	extractedKeys := []*core.Jwk{
		{
			KTY:     "test-kty",
			Use:     "test-use",
			Alg:     "test-alg",
			KID:     storedKeys[0].ID.String(),
			Payload: json.RawMessage(`{"message":"hello world"}`),
		},
		{
			KTY:     "test-kty",
			Use:     "test-use",
			Alg:     "test-alg",
			KID:     storedKeys[1].ID.String(),
			Payload: json.RawMessage(`{"message":"hello world"}`),
		},
	}
	request := &core.JwkSearchRequest{Usage: "test-usage", Private: true}

	testCases := []struct {
		name string

		request *core.JwkSearchRequest

		daoSearchMock      *daoSearchMock
		serviceExtractMock []*serviceExtractMock

		expect    []*core.Jwk
		expectErr error
	}{
		{
			name:    "Success",
			request: request,

			daoSearchMock: &daoSearchMock{resp: storedKeys},
			serviceExtractMock: []*serviceExtractMock{
				{resp: extractedKeys[0]},
				{resp: extractedKeys[1]},
			},

			expect: extractedKeys,
		},
		{
			name:    "Success/Empty",
			request: request,

			daoSearchMock: &daoSearchMock{resp: []*dao.Jwk{}},

			expect: []*core.Jwk{},
		},
		{
			name:    "Error/Extract",
			request: request,

			daoSearchMock: &daoSearchMock{resp: storedKeys},
			serviceExtractMock: []*serviceExtractMock{
				{resp: extractedKeys[0]},
				{err: errFoo},
			},

			expectErr: errFoo,
		},
		{
			name:    "Error/Search",
			request: request,

			daoSearchMock: &daoSearchMock{err: errFoo},

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
					Exec(mock.Anything, &dao.JwkSearchRequest{Usage: testCase.request.Usage}).
					Return(testCase.daoSearchMock.resp, testCase.daoSearchMock.err)
			}

			if testCase.serviceExtractMock != nil {
				require.GreaterOrEqual(t, len(testCase.serviceExtractMock), len(testCase.daoSearchMock.resp))

				for i, extracted := range testCase.serviceExtractMock {
					extractRequest := &core.JwkExtractRequest{
						Jwk:     testCase.daoSearchMock.resp[i],
						Private: testCase.request.Private,
					}

					serviceExtract.EXPECT().
						Exec(mock.Anything, extractRequest).
						Return(extracted.resp, extracted.err).
						Once()
				}
			}

			service := core.NewJwkSearch(daoSearch, serviceExtract)

			res, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)
		})
	}

	// The real extraction, with the request of the REST handlers, which leave Private false. A
	// symmetric key has only private material, so the search fails before deserializing any of it.
	t.Run("Error/SymmetricKeyWithoutPrivateAuthorization", func(t *testing.T) {
		t.Parallel()

		symmetricJwk := &jwa.JWK{
			KTY: "oct", Alg: "HS256", KID: "00000000-0000-0000-0000-000000000001",
			Payload: []byte(`{"k":"secret-octets"}`),
		}
		symmetric := &dao.Jwk{
			ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Usage: "auth",
			// No public key: the shape a symmetric algorithm is stored in.
			PrivateKey: mustEncryptBase64Value(t, symmetricJwk),
		}

		daoSearch := coremocks.NewMockJwkSearchDao(t)
		daoSearch.EXPECT().Exec(mock.Anything, mock.Anything).Return([]*dao.Jwk{symmetric}, nil)

		service := core.NewJwkSearch(daoSearch, core.NewJwkExtract(nil))

		keys, err := service.Exec(t.Context(), &core.JwkSearchRequest{Usage: "auth"})
		require.ErrorIs(t, err, core.ErrJwkExtractNoPublicKey)
		require.Nil(t, keys)
	})
}
