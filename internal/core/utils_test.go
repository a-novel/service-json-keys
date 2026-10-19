package core_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

// supportedAlgorithms lists every signing algorithm a usage may configure.
var supportedAlgorithms = []jwa.Alg{
	jwa.Ed25519,
	jwa.EdDSA, //nolint:staticcheck // The legacy label is under test.
	jwa.ES256,
	jwa.ES384,
	jwa.ES512,
	jwa.RS256,
	jwa.RS384,
	jwa.RS512,
	jwa.PS256,
	jwa.PS384,
	jwa.PS512,
	jwa.MLDSA44,
	jwa.MLDSA65,
	jwa.MLDSA87,
}

// mustEncryptBase64Value encrypts data with the master key held by ctx, in the form private keys
// are stored.
func mustEncryptBase64Value(ctx context.Context, t *testing.T, data any) string {
	t.Helper()

	res, err := lib.EncryptMasterKey(ctx, data)
	if err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(res)
}

// mustSerializeBase64Value serializes data in the form public keys are stored.
func mustSerializeBase64Value(t *testing.T, data any) string {
	t.Helper()

	res, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(res)
}
