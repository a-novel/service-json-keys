package configtest

import (
	"time"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

const (
	// testTokenTTL is how long a token signed under [TestKeys] stays valid.
	testTokenTTL = 24 * time.Hour
	// testTokenLeeway is the clock skew [TestKeys] tolerates.
	testTokenLeeway = 5 * time.Minute
)

// SupportedAlgorithms lists every signing algorithm a usage may configure.
var SupportedAlgorithms = []jwa.Alg{
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

// TestKeys configures test-usage to sign with alg, under the key lifetimes of key and a fixed token
// target.
func TestKeys(alg jwa.Alg, key config.JwkKey) map[string]*config.Jwk {
	return map[string]*config.Jwk{
		"test-usage": {
			Alg: alg,
			Key: key,
			Token: config.JwkToken{
				TTL:      testTokenTTL,
				Issuer:   "test-issuer",
				Audience: "test-audience",
				Subject:  "test-subject",
				Leeway:   testTokenLeeway,
			},
		},
	}
}

// SupportedAlgorithmKeys configures one usage per supported algorithm, named after it.
func SupportedAlgorithmKeys() map[string]*config.Jwk {
	keys := make(map[string]*config.Jwk, len(SupportedAlgorithms))
	for _, alg := range SupportedAlgorithms {
		keys[string(alg)] = &config.Jwk{Alg: alg}
	}

	return keys
}
