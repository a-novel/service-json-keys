package jwkconfig

import (
	"errors"

	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jws"
)

// ErrJwkPresetUnknownAlgorithm is returned when a key configuration references an algorithm with no
// signing or verification plugin. Only asymmetric algorithms are supported because symmetric secrets
// have no public half to publish on the REST surface.
var ErrJwkPresetUnknownAlgorithm = errors.New("unknown jwk algorithm")

// JwsPresetsEcdsa maps ECDSA algorithm identifiers to their JWS signing/verification presets.
var JwsPresetsEcdsa = map[jwa.Alg]jws.ECDSAPreset{
	jwa.ES256: jws.ES256,
	jwa.ES384: jws.ES384,
	jwa.ES512: jws.ES512,
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
