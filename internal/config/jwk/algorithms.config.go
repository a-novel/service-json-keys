package jwk

import (
	"github.com/a-novel-kit/jwt/v2/jwa"
	jwtjwk "github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"
)

// JwkPresetsEcdsa maps ECDSA algorithm identifiers to their JWK generation presets.
var JwkPresetsEcdsa = map[jwa.Alg]jwtjwk.ECDSAPreset{
	jwa.ES256: jwtjwk.ES256,
	jwa.ES384: jwtjwk.ES384,
	jwa.ES512: jwtjwk.ES512,
}

// JwsPresetsEcdsa maps ECDSA algorithm identifiers to their JWS signing/verification presets.
var JwsPresetsEcdsa = map[jwa.Alg]jws.ECDSAPreset{
	jwa.ES256: jws.ES256,
	jwa.ES384: jws.ES384,
	jwa.ES512: jws.ES512,
}

// JwkPresetsRsa maps RSA algorithm identifiers to their JWK generation presets (covers both PKCS#1 and PSS variants).
var JwkPresetsRsa = map[jwa.Alg]jwtjwk.RSAPreset{
	jwa.RS256: jwtjwk.RS256,
	jwa.RS384: jwtjwk.RS384,
	jwa.RS512: jwtjwk.RS512,
	jwa.PS256: jwtjwk.PS256,
	jwa.PS384: jwtjwk.PS384,
	jwa.PS512: jwtjwk.PS512,
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
