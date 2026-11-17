package core_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

func TestJwkSelect(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoSelectMock struct {
		resp *dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *core.Jwk
		err  error
	}

	storedKey := &dao.Jwk{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		PrivateKey: "cHJpdmF0ZS1rZXktMg",
		PublicKey:  new("cHVibGljLWtleS0y"),
		Usage:      "test-usage",
		CreatedAt:  time.Now().Add(-time.Hour),
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	extractedKey := &core.Jwk{
		KTY:     "test-kty",
		Use:     "test-use",
		Alg:     "test-alg",
		KID:     storedKey.ID.String(),
		Payload: json.RawMessage(`{"message":"hello world"}`),
	}
	request := &core.JwkSelectRequest{ID: storedKey.ID}

	testCases := []struct {
		name string

		request *core.JwkSelectRequest

		daoSelectMock      *daoSelectMock
		serviceExtractMock *serviceExtractMock

		expect    *core.Jwk
		expectErr error
	}{
		{
			name:    "Success",
			request: request,

			daoSelectMock:      &daoSelectMock{resp: storedKey},
			serviceExtractMock: &serviceExtractMock{resp: extractedKey},

			expect: extractedKey,
		},
		{
			name:    "Error/Extract",
			request: request,

			daoSelectMock:      &daoSelectMock{resp: storedKey},
			serviceExtractMock: &serviceExtractMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name:    "Error/NotFound",
			request: request,

			daoSelectMock: &daoSelectMock{err: dao.ErrJwkNotFound},

			expectErr: core.ErrJwkNotFound,
		},
		{
			name:    "Error/Select",
			request: request,

			daoSelectMock: &daoSelectMock{err: errFoo},

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
					Exec(mock.Anything, &dao.JwkSelectRequest{ID: testCase.request.ID}).
					Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
			}

			if testCase.serviceExtractMock != nil {
				serviceExtract.EXPECT().
					Exec(mock.Anything, &core.JwkExtractRequest{Jwk: testCase.daoSelectMock.resp}).
					Return(testCase.serviceExtractMock.resp, testCase.serviceExtractMock.err).
					Once()
			}

			service := core.NewJwkSelect(daoSelect, serviceExtract)

			res, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)
		})
	}
}
