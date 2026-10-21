package core_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

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
