package core_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
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

// testKeys configures test-usage to sign with alg, under the key lifetimes of key and a fixed token
// target.
func testKeys(alg jwa.Alg, key config.JwkKey) map[string]*config.Jwk {
	return map[string]*config.Jwk{
		"test-usage": {
			Alg: alg,
			Key: key,
			Token: config.JwkToken{
				TTL:      24 * time.Hour,
				Issuer:   "test-issuer",
				Audience: "test-audience",
				Subject:  "test-subject",
				Leeway:   5 * time.Minute,
			},
		},
	}
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
