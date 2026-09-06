package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-json-keys/v2/internal/config"
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

// JwkGenerators is the registry of key generators keyed by algorithm. JwkGen.Exec uses this
// to look up the correct generator for a given usage's configured algorithm.
var JwkGenerators = map[jwa.Alg]JwkGenAny{
	jwa.EdDSA: JwkGeneratorEd25519,
	jwa.ES256: JwkGeneratorEs(jwa.ES256),
	jwa.ES384: JwkGeneratorEs(jwa.ES384),
	jwa.ES512: JwkGeneratorEs(jwa.ES512),
	jwa.RS256: JwkGeneratorRsa(jwa.RS256),
	jwa.RS384: JwkGeneratorRsa(jwa.RS384),
	jwa.RS512: JwkGeneratorRsa(jwa.RS512),
	jwa.PS256: JwkGeneratorRsa(jwa.PS256),
	jwa.PS384: JwkGeneratorRsa(jwa.PS384),
	jwa.PS512: JwkGeneratorRsa(jwa.PS512),
}

// JwkGeneratorEd25519 generates an Ed25519 private/public key pair.
func JwkGeneratorEd25519() (*JwkGeneratorResult, error) {
	priv, pub, err := jwk.GenerateED25519()
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

// JwkGeneratorEs returns a generator for the given ECDSA algorithm.
func JwkGeneratorEs(alg jwa.Alg) JwkGenAny {
	return func() (*JwkGeneratorResult, error) {
		var (
			preset jwk.ECDSAPreset
			ok     bool
		)

		if preset, ok = JwkPresetsEcdsa[alg]; !ok {
			return nil, fmt.Errorf("%w (ecdsa): %s", ErrJwkPresetUnknown, alg)
		}

		priv, pub, err := jwk.GenerateECDSA(preset)
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
}

// JwkGeneratorRsa returns a generator for the given RSA algorithm (covers both PKCS#1 and PSS).
func JwkGeneratorRsa(alg jwa.Alg) JwkGenAny {
	return func() (*JwkGeneratorResult, error) {
		var (
			preset jwk.RSAPreset
			ok     bool
		)

		if preset, ok = JwkPresetsRsa[alg]; !ok {
			return nil, fmt.Errorf("%w (rsa): %s", ErrJwkPresetUnknown, alg)
		}

		priv, pub, err := jwk.GenerateRSA(preset)
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
}

// JwkPrivateSource provides the private JWKs used by [NewJwkProducers].
type JwkPrivateSource interface {
	// SearchKeys returns the private JWKs registered for usage.
	SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error)
}

// JwkPublicSource provides the public JWKs used by [NewJwkRecipients].
type JwkPublicSource interface {
	// SearchKeys returns the public JWKs registered for usage.
	SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error)
}

func newJwkSource(
	searchKeys func(context.Context, string) ([]*jwa.JWK, error),
	usage string,
	sourceConfig jwk.SourceConfig,
) *jwk.Source {
	sourceConfig.Fetch = func(ctx context.Context) ([]*jwa.JWK, error) {
		return searchKeys(ctx, usage)
	}

	return jwk.NewSource(sourceConfig)
}

// JwkProducers maps each key usage to the set of JWT producer plugins used for signing tokens
// under that usage.
type JwkProducers map[string][]jwt.ProducerPlugin

// NewJwkProducers builds cached signing plugins from private keys for every configured usage.
func NewJwkProducers(
	source JwkPrivateSource,
	keys map[string]*config.Jwk,
) (JwkProducers, error) {
	output := make(JwkProducers)

	for usage, keyConfig := range keys {
		keySource := newJwkSource(source.SearchKeys, usage, jwk.SourceConfig{
			CacheDuration: keyConfig.Key.Cache,
		})

		var signer jwt.ProducerPlugin

		switch keyConfig.Alg {
		case jwa.EdDSA:
			signer = jws.NewSourcedED25519Signer(keySource)
		case jwa.ES256, jwa.ES384, jwa.ES512:
			ecdsaPreset, ok := JwsPresetsEcdsa[keyConfig.Alg]
			if !ok {
				return nil, fmt.Errorf("%w (ecdsa) for usage: %s", ErrJwkPresetUnknown, usage)
			}

			signer = jws.NewSourcedECDSASigner(keySource, ecdsaPreset)
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
			rsaPreset, ok := JwsPresetsRsa[keyConfig.Alg]
			if !ok {
				return nil, fmt.Errorf("%w (rsa) for usage: %s", ErrJwkPresetUnknown, usage)
			}

			signer = jws.NewSourcedRSASigner(keySource, rsaPreset)
		default:
			return nil, fmt.Errorf("%w: %s", ErrJwkPresetUnknownAlgorithm, keyConfig.Alg)
		}

		output[usage] = []jwt.ProducerPlugin{signer}
	}

	return output, nil
}

// JwkRecipients maps each key usage to the set of JWT recipient plugins used for verifying tokens
// under that usage.
type JwkRecipients map[string][]jwt.RecipientPlugin

// NewJwkRecipients builds cached verification plugins from public keys for every configured usage.
func NewJwkRecipients(
	source JwkPublicSource,
	keys map[string]*config.Jwk,
) (JwkRecipients, error) {
	output := make(JwkRecipients)

	for usage, keyConfig := range keys {
		keySource := newJwkSource(source.SearchKeys, usage, jwk.SourceConfig{
			CacheDuration: keyConfig.Key.Cache,
			// A rotated signing key can appear before the verifier's normal cache refresh.
			// An unknown key ID triggers one rate-limited refetch so verification can continue.
			RefreshOnUnknownKeyID: true,
			UnknownKeyIDInterval:  keyConfig.Key.UnknownKeyIDInterval,
		})

		var recipient jwt.RecipientPlugin

		switch keyConfig.Alg {
		case jwa.EdDSA:
			recipient = jws.NewSourcedED25519Verifier(keySource)
		case jwa.ES256, jwa.ES384, jwa.ES512:
			ecdsaPreset, ok := JwsPresetsEcdsa[keyConfig.Alg]
			if !ok {
				return nil, fmt.Errorf("%w (ecdsa) for usage: %s", ErrJwkPresetUnknown, usage)
			}

			recipient = jws.NewSourcedECDSAVerifier(keySource, ecdsaPreset)
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
			rsaPreset, ok := JwsPresetsRsa[keyConfig.Alg]
			if !ok {
				return nil, fmt.Errorf("%w (rsa) for usage: %s", ErrJwkPresetUnknown, usage)
			}

			recipient = jws.NewSourcedRSAVerifier(keySource, rsaPreset)
		default:
			return nil, fmt.Errorf("%w: %s", ErrJwkPresetUnknownAlgorithm, keyConfig.Alg)
		}

		output[usage] = []jwt.RecipientPlugin{recipient}
	}

	return output, nil
}
