package services_test

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

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/config"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	testutils "github.com/a-novel/service-json-keys/internal/lib/test"
	"github.com/a-novel/service-json-keys/internal/services"
	servicesmocks "github.com/a-novel/service-json-keys/internal/services/mocks"
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

func TestGenerateKeys(t *testing.T) { //nolint:paralleltest
	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type repositorySearchMock struct {
		resp []*dao.Jwk
		err  error
	}

	type repositoryInsertMock struct {
		resp *dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *services.Jwk
		err  error
	}

	type testCaseDef struct {
		name string

		request *services.JwkGenRequest

		repositorySearchMock *repositorySearchMock
		repositoryInsertMock *repositoryInsertMock
		serviceExtractMock   *serviceExtractMock
		keys                 map[string]*config.Jwk

		expect    *services.Jwk
		expectErr error
	}

	testCases := []testCaseDef{
		{
			name: "Success/OldKeys",

			request: &services.JwkGenRequest{
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

			repositorySearchMock: &repositorySearchMock{
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

			repositoryInsertMock: &repositoryInsertMock{
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
				resp: &services.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: jwa.EdDSA,
						KID: "00000000-0000-0000-0000-000000000001",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &services.Jwk{
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

			request: &services.JwkGenRequest{
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

			repositorySearchMock: &repositorySearchMock{
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
				resp: &services.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: jwa.EdDSA,
						KID: "00000000-0000-0000-0000-000000000002",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &services.Jwk{
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

			request: &services.JwkGenRequest{
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

			repositorySearchMock: &repositorySearchMock{
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

			repositoryInsertMock: &repositoryInsertMock{
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

			request: &services.JwkGenRequest{
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

			repositorySearchMock: &repositorySearchMock{
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

			repositoryInsertMock: &repositoryInsertMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Search",

			request: &services.JwkGenRequest{
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

			repositorySearchMock: &repositorySearchMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, alg := range []jwa.Alg{
		jwa.EdDSA,
		jwa.HS256,
		jwa.HS384,
		jwa.HS512,
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

			request: &services.JwkGenRequest{
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

			repositorySearchMock: &repositorySearchMock{},

			repositoryInsertMock: &repositoryInsertMock{
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
				resp: &services.Jwk{
					JWKCommon: jwa.JWKCommon{
						KTY: "test-kty",
						Use: "test-use",
						Alg: alg,
						KID: "00000000-0000-0000-0000-000000000001",
					},
					Payload: json.RawMessage(`{"message":"hello world"}`),
				},
			},

			expect: &services.Jwk{
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

	for _, testCase := range testCases { //nolint:paralleltest
		t.Run(testCase.name, func(t *testing.T) {
			repositorySearch := servicesmocks.NewMockJwkGenRepositorySearch(t)
			repositoryInsert := servicesmocks.NewMockJwkGenRepositoryInsert(t)
			serviceExtract := servicesmocks.NewMockJwkGenServiceExtract(t)

			if testCase.repositorySearchMock != nil {
				repositorySearch.EXPECT().
					Exec(mock.Anything, &dao.JwkSearchRequest{
						Usage: testCase.request.Usage,
					}).
					Return(testCase.repositorySearchMock.resp, testCase.repositorySearchMock.err)
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
				_, err = checkGeneratedPrivateKey(ctx, t, request.PrivateKey)
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

			if testCase.repositoryInsertMock != nil {
				repositoryInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(checkInsertData)).
					Return(testCase.repositoryInsertMock.resp, testCase.repositoryInsertMock.err)
			}

			if testCase.serviceExtractMock != nil {
				var keyToExtract *dao.Jwk

				if testCase.repositoryInsertMock != nil {
					keyToExtract = testCase.repositoryInsertMock.resp
				} else if testCase.repositorySearchMock != nil {
					keyToExtract = testCase.repositorySearchMock.resp[0]
				} else {
					t.Fatal(
						"expected repositorySearchMock or repositoryInsertMock to be defined for serviceExtractMock",
					)
				}

				serviceExtract.EXPECT().
					Exec(mock.Anything, &services.JwkExtractRequest{Jwk: keyToExtract, Private: true}).
					Return(testCase.serviceExtractMock.resp, testCase.serviceExtractMock.err)
			}

			service := services.NewJwkGen(
				repositorySearch,
				repositoryInsert,
				serviceExtract,
				testCase.keys,
			)

			key, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, key)

			repositorySearch.AssertExpectations(t)
			repositoryInsert.AssertExpectations(t)
			serviceExtract.AssertExpectations(t)
		})
	}
}
