package core_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	testutils "github.com/a-novel/service-json-keys/v2/internal/lib/test"
)

func checkGeneratedPrivateKey(ctx context.Context, t *testing.T, key string) (*jwa.JWK, error) {
	t.Helper()

	// Decode base64 value.
	decoded, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// Decrypt.
	var decrypted jwa.JWK

	err = lib.DecryptMasterKey(ctx, decoded, &decrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt key: %w", err)
	}

	return &decrypted, nil
}

func checkGeneratedPublicKey(t *testing.T, key string) (*jwa.JWK, error) {
	t.Helper()

	// Decode base64 value.
	decoded, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// Unmarshal.
	var deserialized jwa.JWK

	err = deserialized.UnmarshalJSON(decoded)
	if err != nil {
		return nil, fmt.Errorf("unmarshal key: %w", err)
	}

	return &deserialized, nil
}

func TestJwkGen(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type daoSearchMock struct {
		resp []*dao.Jwk
		err  error
	}

	type daoInsertMock struct {
		resp *dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *core.Jwk
		err  error
	}

	type testCaseDef struct {
		name string

		request *core.JwkGenRequest

		daoSearchMock      *daoSearchMock
		daoInsertMock      *daoInsertMock
		serviceExtractMock *serviceExtractMock
		keys               map[string]*config.Jwk

		expect    *core.Jwk
		expectErr error
	}

	testCases := []testCaseDef{
		{
			name: "Success/OldKeys",

			request: &core.JwkGenRequest{
				Usage: "test-usage",
			},

			keys: map[string]*config.Jwk{
				"test-usage": {
					Alg: jwa.EdDSA,
					Key: config.JwkKey{
						TTL:      24 * time.Hour,
						Rotation: 12 * time.Hour,
						Cache:    6 * time.Hour,
					},
				},
			},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-13 * time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			daoInsertMock: &daoInsertMock{
				resp: &dao.Jwk{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  time.Now(),
					ExpiresAt:  time.Now().Add(24 * time.Hour),
				},
			},

			serviceExtractMock: &serviceExtractMock{
				resp: &core.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: jwa.EdDSA,
						KID: "00000000-0000-0000-0000-000000000001",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &core.Jwk{
				JWKCommon: jwa.JWKCommon{
					KTY: "test-kty",
					Use: "test-use",
					Alg: jwa.EdDSA,
					KID: "00000000-0000-0000-0000-000000000001",
				},
				Payload: json.RawMessage(`{"message":"hello world"}`),
			},
		},
		{
			name: "Success/RecentKeys",

			request: &core.JwkGenRequest{
				Usage: "test-usage",
			},

			keys: map[string]*config.Jwk{
				"test-usage": {
					Alg: jwa.EdDSA,
					Key: config.JwkKey{
						TTL:      24 * time.Hour,
						Rotation: 12 * time.Hour,
						Cache:    6 * time.Hour,
					},
				},
			},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-11 * time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			serviceExtractMock: &serviceExtractMock{
				resp: &core.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: jwa.EdDSA,
						KID: "00000000-0000-0000-0000-000000000002",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &core.Jwk{
				JWKCommon: jwa.JWKCommon{
					KTY: "test-kty",
					Use: "test-use",
					Alg: jwa.EdDSA,
					KID: "00000000-0000-0000-0000-000000000002",
				},
				Payload: json.RawMessage(`{"message":"hello world"}`),
			},
		},
		{
			name: "Error/Extract",

			request: &core.JwkGenRequest{
				Usage: "test-usage",
			},

			keys: map[string]*config.Jwk{
				"test-usage": {
					Alg: jwa.EdDSA,
					Key: config.JwkKey{
						TTL:      24 * time.Hour,
						Rotation: 12 * time.Hour,
						Cache:    6 * time.Hour,
					},
				},
			},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-13 * time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			daoInsertMock: &daoInsertMock{
				resp: &dao.Jwk{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  time.Now(),
					ExpiresAt:  time.Now().Add(24 * time.Hour),
				},
			},

			serviceExtractMock: &serviceExtractMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Insert",

			request: &core.JwkGenRequest{
				Usage: "test-usage",
			},

			keys: map[string]*config.Jwk{
				"test-usage": {
					Alg: jwa.EdDSA,
					Key: config.JwkKey{
						TTL:      24 * time.Hour,
						Rotation: 12 * time.Hour,
						Cache:    6 * time.Hour,
					},
				},
			},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      "test-usage",
						CreatedAt:  time.Now().Add(-13 * time.Hour),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			daoInsertMock: &daoInsertMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Search",

			request: &core.JwkGenRequest{
				Usage: "test-usage",
			},

			keys: map[string]*config.Jwk{
				"test-usage": {
					Alg: jwa.EdDSA,
					Key: config.JwkKey{
						TTL:      24 * time.Hour,
						Rotation: 12 * time.Hour,
						Cache:    6 * time.Hour,
					},
				},
			},

			daoSearchMock: &daoSearchMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ConfigNotFound",

			request: &core.JwkGenRequest{
				Usage: "test-usage",
			},

			keys: map[string]*config.Jwk{},

			daoSearchMock: &daoSearchMock{
				resp: []*dao.Jwk{},
			},

			expectErr: core.ErrConfigNotFound,
		},
	}

	for _, alg := range []jwa.Alg{
		jwa.EdDSA,
		jwa.RS256,
		jwa.RS384,
		jwa.RS512,
		jwa.PS256,
		jwa.PS384,
		jwa.PS512,
		jwa.ES256,
		jwa.ES384,
		jwa.ES512,
	} {
		testCases = append(testCases, testCaseDef{
			name: "Success/" + string(alg),

			request: &core.JwkGenRequest{
				Usage: "test-usage",
			},

			keys: map[string]*config.Jwk{
				"test-usage": {
					Alg: alg,
					Key: config.JwkKey{
						TTL:      24 * time.Hour,
						Rotation: 12 * time.Hour,
						Cache:    6 * time.Hour,
					},
				},
			},

			daoSearchMock: &daoSearchMock{},

			daoInsertMock: &daoInsertMock{
				resp: &dao.Jwk{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  time.Now(),
					ExpiresAt:  time.Now().Add(24 * time.Hour),
				},
			},

			serviceExtractMock: &serviceExtractMock{
				resp: &core.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: alg,
						KID: "00000000-0000-0000-0000-000000000001",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &core.Jwk{
				JWKCommon: jwa.JWKCommon{
					KTY: "test-kty",
					Use: "test-use",
					Alg: alg,
					KID: "00000000-0000-0000-0000-000000000001",
				},
				Payload: json.RawMessage(`{"message":"hello world"}`),
			},
		})
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSearch := coremocks.NewMockJwkGenDaoSearch(t)
			daoInsert := coremocks.NewMockJwkGenDaoInsert(t)
			serviceExtract := coremocks.NewMockJwkGenServiceExtract(t)

			if testCase.daoSearchMock != nil {
				daoSearch.EXPECT().
					Exec(mock.Anything, &dao.JwkSearchRequest{
						Usage: testCase.request.Usage,
					}).
					Return(testCase.daoSearchMock.resp, testCase.daoSearchMock.err)
			}

			checkInsertData := func(request *dao.JwkInsertRequest) bool {
				if request == nil {
					t.Errorf("request is nil")

					return false
				}

				if request.ID == uuid.Nil {
					t.Error("expected KID to be set, got nil")

					return false
				}

				// Ensure private key is encrypted.
				_, err := checkGeneratedPrivateKey(ctx, t, request.PrivateKey)
				if err != nil {
					t.Errorf("checking private key: %s", err)

					return false
				}

				if request.PublicKey != nil {
					_, err = checkGeneratedPublicKey(t, *request.PublicKey)
					if err != nil {
						t.Errorf("checking public key: %s", err)

						return false
					}
				}

				if request.Expiration.IsZero() {
					t.Error("expected expiration to be set, got zero")

					return false
				}

				if request.Expiration.Before(request.Now) {
					t.Error("expected expiration to be after creation date")

					return false
				}

				return true
			}

			if testCase.daoInsertMock != nil {
				daoInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(checkInsertData)).
					Return(testCase.daoInsertMock.resp, testCase.daoInsertMock.err)
			}

			if testCase.serviceExtractMock != nil {
				var keyToExtract *dao.Jwk

				if testCase.daoInsertMock != nil {
					keyToExtract = testCase.daoInsertMock.resp
				} else if testCase.daoSearchMock != nil {
					keyToExtract = testCase.daoSearchMock.resp[0]
				} else {
					t.Fatal(
						"expected daoSearchMock or daoInsertMock to be defined for serviceExtractMock",
					)
				}

				serviceExtract.EXPECT().
					Exec(mock.Anything, &core.JwkExtractRequest{Jwk: keyToExtract, Private: true}).
					Return(testCase.serviceExtractMock.resp, testCase.serviceExtractMock.err)
			}

			service := core.NewJwkGen(
				daoSearch,
				daoInsert,
				serviceExtract,
				testCase.keys,
			)

			key, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, key)

			daoSearch.AssertExpectations(t)
			daoInsert.AssertExpectations(t)
			serviceExtract.AssertExpectations(t)
		})
	}
}
