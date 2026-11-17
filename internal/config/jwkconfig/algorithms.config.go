package jwkconfig

import (
	"errors"

	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"
)

var (
	// ErrJwkPresetUnknown is returned when a requested algorithm has no corresponding preset entry.
	ErrJwkPresetUnknown = errors.New("unknown jwk preset")
	// ErrJwkPresetUnknownAlgorithm is returned when a key configuration references an algorithm
	// with no signing or verification plugin. Only asymmetric algorithms are supported because
	// symmetric secrets have no public half to publish on the REST surface.
	ErrJwkPresetUnknownAlgorithm = errors.New("unknown jwk algorithm")
)

// JwkPresetsEcdsa maps ECDSA algorithm identifiers to their JWK generation presets.
var JwkPresetsEcdsa = map[jwa.Alg]jwk.ECDSAPreset{
	jwa.ES256: jwk.ES256,
	jwa.ES384: jwk.ES384,
	jwa.ES512: jwk.ES512,
}

// JwsPresetsEcdsa maps ECDSA algorithm identifiers to their JWS signing/verification presets.
var JwsPresetsEcdsa = map[jwa.Alg]jws.ECDSAPreset{
	jwa.ES256: jws.ES256,
	jwa.ES384: jws.ES384,
	jwa.ES512: jws.ES512,
}

// JwkPresetsRsa maps RSA algorithm identifiers to their JWK generation presets (covers both PKCS#1 and PSS variants).
var JwkPresetsRsa = map[jwa.Alg]jwk.RSAPreset{
	jwa.RS256: jwk.RS256,
	jwa.RS384: jwk.RS384,
	jwa.RS512: jwk.RS512,
	jwa.PS256: jwk.PS256,
	jwa.PS384: jwk.PS384,
	jwa.PS512: jwk.PS512,
}

// JwsPresetsRsa maps RSA algorithm identifiers to their JWS signing/verification presets. In jwt v2
// the RS* and PS* presets share one RSAPreset type, so PKCS#1 and PSS live in the same map.
var JwsPresetsRsa = map[jwa.Alg]jws.RSAPreset{
	jwa.RS256: jws.RS256,
	jwa.RS384: jws.RS384,
	jwa.RS512: jws.RS512,
	jwa.PS256: jws.PS256,
	jwa.PS384: jws.PS384,
	jwa.PS512: jws.PS512,
}

// JwkPresetsMLDSA maps ML-DSA algorithm identifiers to their JWK generation presets. The JWS plugins
// need no preset: an ML-DSA key's parameter set fixes its algorithm.
var JwkPresetsMLDSA = map[jwa.Alg]jwk.MLDSAPreset{
	jwa.MLDSA44: jwk.MLDSA44,
	jwa.MLDSA65: jwk.MLDSA65,
	jwa.MLDSA87: jwk.MLDSA87,
}
