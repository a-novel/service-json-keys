package core

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
)

// ErrJwkPresetUnknownAlgorithm is returned when a key configuration references an algorithm with no
// signing plugin.
var ErrJwkPresetUnknownAlgorithm = jwkconfig.ErrJwkPresetUnknownAlgorithm

// JwkGeneratorResult contains the key material and identifiers produced for one asymmetric key pair.
type JwkGeneratorResult struct {
	// PrivateKey contains the private key material.
	PrivateKey any
	// PublicKey contains the matching public key material.
	PublicKey any
	// PrivateKID identifies PrivateKey.
	PrivateKID string
	// PublicKID identifies PublicKey.
	PublicKID string
}

// JwkGenAny is the common generator signature used by [JwkGenerators].
type JwkGenAny func() (*JwkGeneratorResult, error)

// JwkGenerators is the registry of key generators keyed by algorithm. [JwkGen.Exec] uses it to
// look up the generator of a usage's configured algorithm.
var JwkGenerators = map[jwa.Alg]JwkGenAny{
	jwa.EdDSA:   JwkGeneratorEd25519(jwa.EdDSA), //nolint:staticcheck // A configured legacy label is honored.
	jwa.Ed25519: JwkGeneratorEd25519(jwa.Ed25519),
	jwa.ES256:   JwkGeneratorEs(jwk.ES256),
	jwa.ES384:   JwkGeneratorEs(jwk.ES384),
	jwa.ES512:   JwkGeneratorEs(jwk.ES512),
	jwa.RS256:   JwkGeneratorRsa(jwk.RS256),
	jwa.RS384:   JwkGeneratorRsa(jwk.RS384),
	jwa.RS512:   JwkGeneratorRsa(jwk.RS512),
	jwa.PS256:   JwkGeneratorRsa(jwk.PS256),
	jwa.PS384:   JwkGeneratorRsa(jwk.PS384),
	jwa.PS512:   JwkGeneratorRsa(jwk.PS512),
	jwa.MLDSA44: JwkGeneratorMLDSA(jwk.MLDSA44),
	jwa.MLDSA65: JwkGeneratorMLDSA(jwk.MLDSA65),
	jwa.MLDSA87: JwkGeneratorMLDSA(jwk.MLDSA87),
}

// JwkGeneratorEd25519 returns a generator for Ed25519 key pairs labeled with alg: "Ed25519", or the
// "EdDSA" identifier RFC 9864 deprecates. Verifiers on jwt v2.2.x recognize only "EdDSA" keys.
func JwkGeneratorEd25519(alg jwa.Alg) JwkGenAny {
	return func() (*JwkGeneratorResult, error) {
		priv, pub, err := jwk.GenerateED25519()
		if err != nil {
			return nil, err
		}

		priv.Alg, pub.Alg = alg, alg

		return newJwkGeneratorResult(priv, pub, nil)
	}
}

// JwkGeneratorEs returns a generator for the ECDSA algorithm of preset.
func JwkGeneratorEs(preset jwk.ECDSAPreset) JwkGenAny {
	return func() (*JwkGeneratorResult, error) {
		return newJwkGeneratorResult(jwk.GenerateECDSA(preset))
	}
}

// JwkGeneratorRsa returns a generator for the RSA algorithm of preset, PKCS#1 or PSS.
func JwkGeneratorRsa(preset jwk.RSAPreset) JwkGenAny {
	return func() (*JwkGeneratorResult, error) {
		return newJwkGeneratorResult(jwk.GenerateRSA(preset))
	}
}

// JwkGeneratorMLDSA returns a generator for the post-quantum ML-DSA algorithm of preset.
func JwkGeneratorMLDSA(preset jwk.MLDSAPreset) JwkGenAny {
	return func() (*JwkGeneratorResult, error) {
		return newJwkGeneratorResult(jwk.GenerateMLDSA(preset))
	}
}

// newJwkGeneratorResult wraps the output of a jwt key-pair generator, so a generator can return
// its call directly.
func newJwkGeneratorResult[Private, Public any](
	priv *jwk.Key[Private], pub *jwk.Key[Public], err error,
) (*JwkGeneratorResult, error) {
	if err != nil {
		return nil, err
	}

	return &JwkGeneratorResult{
		PrivateKey: priv,
		PublicKey:  pub,
		PrivateKID: priv.KID,
		PublicKID:  pub.KID,
	}, nil
}

// JwkProducersServiceSearch is the service dependency of [NewJwkProducers] for fetching private keys.
type JwkProducersServiceSearch interface {
	Exec(ctx context.Context, request *JwkSearchRequest) ([]*Jwk, error)
}

// JwkProducers maps each key usage to the set of JWT producer plugins used for signing tokens
// under that usage.
type JwkProducers map[string][]jwt.ProducerPlugin

// NewJwkProducers builds cached signing plugins from the private keys of every configured usage.
func NewJwkProducers(
	serviceSearch JwkProducersServiceSearch,
	keys map[string]*jwkconfig.Usage,
) (JwkProducers, error) {
	output := make(JwkProducers)

	for usage, keyConfig := range keys {
		// The source caches the usage's keys, so signing reads the database once per cache period.
		keySource := jwk.NewSource(jwk.SourceConfig{
			Fetch: func(ctx context.Context) ([]*jwa.JWK, error) {
				return serviceSearch.Exec(ctx, &JwkSearchRequest{Usage: usage, Private: true})
			},
			CacheDuration:        keyConfig.Key.Cache,
			StaleIfError:         keyConfig.Key.StaleIfError,
			StaleWhileRevalidate: keyConfig.Key.StaleWhileRevalidate,
		})

		signer, err := newJwkSigner(keySource, keyConfig.Alg)
		if err != nil {
			return nil, err
		}

		output[usage] = []jwt.ProducerPlugin{signer}
	}

	return output, nil
}

// newJwkSigner returns the plugin that signs tokens of alg with the keys of keySource.
func newJwkSigner(keySource *jwk.Source, alg jwa.Alg) (jwt.ProducerPlugin, error) {
	if preset, ok := jwkconfig.JwsPresetsEcdsa[alg]; ok {
		return jws.NewSourcedECDSASigner(keySource, preset), nil
	}

	if preset, ok := jwkconfig.JwsPresetsRsa[alg]; ok {
		return jws.NewSourcedRSASigner(keySource, preset), nil
	}

	switch alg {
	// The token label follows the configured algorithm, so a usage left on "EdDSA" stays verifiable
	// by consumers on jwt v2.2.x, which reject "Ed25519".
	case jwa.EdDSA: //nolint:staticcheck // A configured legacy label is honored.
		return jws.NewSourcedEdDSASigner(keySource), nil //nolint:staticcheck // See above.
	case jwa.Ed25519:
		return jws.NewSourcedED25519Signer(keySource), nil
	case jwa.MLDSA44, jwa.MLDSA65, jwa.MLDSA87:
		return jws.NewSourcedMLDSASigner(keySource), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrJwkPresetUnknownAlgorithm, alg)
	}
}
