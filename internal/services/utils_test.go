package services_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwk"

	"github.com/a-novel/service-json-keys/internal/lib"
)

func mustEncryptValue(ctx context.Context, t *testing.T, data any) []byte {
	t.Helper()

	res, err := lib.EncryptMasterKey(ctx, data)
	if err != nil {
		panic(err)
	}

	return res
}

func mustEncryptBase64Value(ctx context.Context, t *testing.T, data any) string {
	t.Helper()

	res := mustEncryptValue(ctx, t, data)

	return base64.RawURLEncoding.EncodeToString(res)
}

func mustSerializeBase64Value(t *testing.T, data any) string {
	t.Helper()

	res, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(res)
}

func generateAuthTokenKeySet(t *testing.T, size int) ([]*jwk.Key[ed25519.PrivateKey], []*jwk.Key[ed25519.PublicKey]) {
	t.Helper()

	privateKeys := make([]*jwk.Key[ed25519.PrivateKey], size)
	publicKeys := make([]*jwk.Key[ed25519.PublicKey], size)

	for i := range size {
		privateKey, publicKey, err := jwk.GenerateED25519()
		require.NoError(t, err)

		privateKeys[i] = privateKey
		publicKeys[i] = publicKey
	}

	return privateKeys, publicKeys
}
