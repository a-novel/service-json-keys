package core_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	testutils "github.com/a-novel/service-json-keys/v2/internal/lib/test"
)

func TestJwkExtract(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	testCases := []struct {
		name string

		request *core.JwkExtractRequest

		expect    *core.Jwk
		expectErr error
	}{
		{
			name: "Success/PublicKey",

			request: &core.JwkExtractRequest{
				Jwk: &dao.Jwk{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),

					PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: []jwa.KeyOp{"test-key-ops"},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000001",
						},
						Payload: []byte(
							`{"value":"private-key-1"}`,
						),
					}),
					PublicKey: lo.ToPtr(mustSerializeBase64Value(t, &jwa.JWK{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: []jwa.KeyOp{"test-key-ops"},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000001",
						},
						Payload: []byte(`{"value":"public-key-1"}`),
					})),
				},
			},

			expect: &core.Jwk{
				JWKCommon: jwa.JWKCommon{
					KTY:    "test-kty",
					Use:    "test-use",
					KeyOps: []jwa.KeyOp{"test-key-ops"},
					Alg:    "test-alg",
					KID:    "00000000-0000-0000-0000-000000000001",
				},
				Payload: []byte(`{"value":"public-key-1"}`),
			},
		},
		{
			name: "Success/SymmetricKey",

			request: &core.JwkExtractRequest{
				Jwk: &dao.Jwk{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),

					PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: []jwa.KeyOp{"test-key-ops"},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000001",
						},
						Payload: []byte(
							`{"value":"private-key-1"}`,
						),
					}),
				},
			},

			expect: &jwa.JWK{
				JWKCommon: jwa.JWKCommon{
					KTY:    "test-kty",
					Use:    "test-use",
					KeyOps: []jwa.KeyOp{"test-key-ops"},
					Alg:    "test-alg",
					KID:    "00000000-0000-0000-0000-000000000001",
				},
				Payload: []byte(`{"value":"private-key-1"}`),
			},
		},
		{
			name: "Success/Private",

			request: &core.JwkExtractRequest{
				Jwk: &dao.Jwk{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),

					PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: []jwa.KeyOp{"test-key-ops"},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000001",
						},
						Payload: []byte(
							`{"value":"private-key-1"}`,
						),
					}),
					PublicKey: lo.ToPtr(mustSerializeBase64Value(t, &jwa.JWK{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: []jwa.KeyOp{"test-key-ops"},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000001",
						},
						Payload: []byte(`{"value":"public-key-1"}`),
					})),
				},

				Private: true,
			},

			expect: &core.Jwk{
				JWKCommon: jwa.JWKCommon{
					KTY:    "test-kty",
					Use:    "test-use",
					KeyOps: []jwa.KeyOp{"test-key-ops"},
					Alg:    "test-alg",
					KID:    "00000000-0000-0000-0000-000000000001",
				},
				Payload: []byte(`{"value":"private-key-1"}`),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := core.NewJwkExtract()

			result, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			// The json.RawMessage in the JWK payload causes trouble with Go comparison,
			// so instead we directly compare the JSON representations.
			jsonExpect, err := json.Marshal(testCase.expect)
			require.NoError(t, err)
			jsonResp, err := json.Marshal(result)
			require.NoError(t, err)

			require.JSONEq(t, string(jsonExpect), string(jsonResp))
		})
	}
}
