package core_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	"github.com/a-novel/service-json-keys/v2/internal/lib/libtest"
)

func TestJwkExtract(t *testing.T) {
	t.Parallel()

	privateJwk := &jwa.JWK{
		KTY:     "test-kty",
		Use:     "test-use",
		KeyOps:  []jwa.KeyOp{"test-key-ops"},
		Alg:     "test-alg",
		KID:     "00000000-0000-0000-0000-000000000001",
		Payload: []byte(`{"value":"private-key-1"}`),
	}
	publicJwk := &jwa.JWK{JWKCommon: privateJwk.JWKCommon, Payload: []byte(`{"value":"public-key-1"}`)}

	// A symmetric key is stored with only private material.
	symmetricKey := &dao.Jwk{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		PrivateKey: mustEncryptBase64Value(t, privateJwk),
	}
	asymmetricKey := &dao.Jwk{
		ID:         symmetricKey.ID,
		PrivateKey: symmetricKey.PrivateKey,
		PublicKey:  new(mustSerializeBase64Value(t, publicJwk)),
	}

	testCases := []struct {
		name string

		request *core.JwkExtractRequest

		expect    *core.Jwk
		expectErr error
	}{
		{
			name:    "Success/PublicKey",
			request: &core.JwkExtractRequest{Jwk: asymmetricKey},
			expect:  publicJwk,
		},
		{
			name:    "Success/Private",
			request: &core.JwkExtractRequest{Jwk: asymmetricKey, Private: true},
			expect:  privateJwk,
		},
		{
			// The gRPC signing path, authorized for private material.
			name:    "Success/SymmetricKeyWithPrivateAuthorization",
			request: &core.JwkExtractRequest{Jwk: symmetricKey, Private: true},
			expect:  privateJwk,
		},
		{
			// A symmetric key has no public half, so a caller without private authorization
			// gets nothing.
			name:      "Error/SymmetricKeyWithoutPrivateAuthorization",
			request:   &core.JwkExtractRequest{Jwk: symmetricKey},
			expectErr: core.ErrJwkExtractNoPublicKey,
		},
	}

	masterKeyCases := []struct {
		name      string
		masterKey *[lib.MasterKeyLength]byte
	}{
		{name: "WithoutMasterKey"},
		{name: "WithMasterKey", masterKey: libtest.MasterKey},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			for _, masterKeyCase := range masterKeyCases {
				t.Run(masterKeyCase.name, func(t *testing.T) {
					t.Parallel()

					service := core.NewJwkExtract(masterKeyCase.masterKey)

					result, err := service.Exec(t.Context(), testCase.request)
					if testCase.request.Private && masterKeyCase.masterKey == nil {
						require.ErrorIs(t, err, lib.ErrInvalidMasterKey)
						require.Nil(t, result)

						return
					}

					require.ErrorIs(t, err, testCase.expectErr)

					jsonExpect, err := json.Marshal(testCase.expect)
					require.NoError(t, err)
					jsonResp, err := json.Marshal(result)
					require.NoError(t, err)

					require.JSONEq(t, string(jsonExpect), string(jsonResp))
				})
			}
		})
	}
}
