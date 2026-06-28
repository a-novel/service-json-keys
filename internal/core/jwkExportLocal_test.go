package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
)

func TestJwkExportLocal(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type sourceMock struct {
		resp []*core.Jwk
		err  error
	}

	testCases := []struct {
		name string

		usage string

		sourceMock *sourceMock

		expect    []*jwa.JWK
		expectErr error
	}{
		{
			name:  "Success",
			usage: "test-usage",

			sourceMock: &sourceMock{
				resp: []*core.Jwk{
					{JWKCommon: jwa.JWKCommon{KID: "kid-1"}},
					{JWKCommon: jwa.JWKCommon{KID: "kid-2"}},
				},
			},

			expect: []*jwa.JWK{
				{JWKCommon: jwa.JWKCommon{KID: "kid-1"}},
				{JWKCommon: jwa.JWKCommon{KID: "kid-2"}},
			},
		},
		{
			name:  "Error/Search",
			usage: "test-usage",

			sourceMock: &sourceMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := coremocks.NewMockJwkExportLocalSource(t)

			if testCase.sourceMock != nil {
				source.EXPECT().
					Exec(mock.Anything, &core.JwkSearchRequest{
						Usage:   testCase.usage,
						Private: true,
					}).
					Return(testCase.sourceMock.resp, testCase.sourceMock.err)
			}

			service := core.NewJwkExportLocal(source)

			result, err := service.SearchKeys(t.Context(), testCase.usage)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			source.AssertExpectations(t)
		})
	}
}
