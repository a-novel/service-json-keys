package core_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/a-novel/service-json-keys/v2/internal/lib"
	"github.com/a-novel/service-json-keys/v2/internal/lib/libtest"
)

// mustEncryptBase64Value encrypts data with the test master key, in the form private keys are
// stored.
func mustEncryptBase64Value(t *testing.T, data any) string {
	t.Helper()

	res, err := lib.EncryptMasterKey(libtest.MasterKey, data)
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
