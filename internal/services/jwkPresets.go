package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

var (
	ErrJwkPresetUnknown          = errors.New("unknown jwk preset")
	ErrJwkPresetUnknownAlgorithm = errors.New("unknown jwk algorithm")
)

var JwkPresetsHMAC = map[jwa.Alg]jwk.HMACPreset{
	jwa.HS256: jwk.HS256,
	jwa.HS384: jwk.HS384,
	jwa.HS512: jwk.HS512,
}

var JwsPresetsHMAC = map[jwa.Alg]jws.HMACPreset{
	jwa.HS256: jws.HS256,
	jwa.HS384: jws.HS384,
	jwa.HS512: jws.HS512,
}

var JwkPresetsEcdsa = map[jwa.Alg]jwk.ECDSAPreset{
	jwa.ES256: jwk.ES256,
	jwa.ES384: jwk.ES384,
	jwa.ES512: jwk.ES512,
}

var JwsPresetsEcdsa = map[jwa.Alg]jws.ECDSAPreset{
	jwa.ES256: jws.ES256,
	jwa.ES384: jws.ES384,
	jwa.ES512: jws.ES512,
}

var JwkPresetsRsa = map[jwa.Alg]jwk.RSAPreset{
	jwa.RS256: jwk.RS256,
	jwa.RS384: jwk.RS384,
	jwa.RS512: jwk.RS512,
	jwa.PS256: jwk.PS256,
	jwa.PS384: jwk.PS384,
	jwa.PS512: jwk.PS512,
}

var JwsPresetsRsa = map[jwa.Alg]jws.RSAPreset{
	jwa.RS256: jws.RS256,
	jwa.RS384: jws.RS384,
	jwa.RS512: jws.RS512,
}

var JwsPresetsRsaPss = map[jwa.Alg]jws.RSAPSSPreset{
	jwa.PS256: jws.PS256,
	jwa.PS384: jws.PS384,
	jwa.PS512: jws.PS512,
}

type JwkGenAny func() (any, any, string, string, error)

var JwkGenerators = map[jwa.Alg]JwkGenAny{
	jwa.EdDSA: JwkGeneratorEd25519,
	jwa.HS256: JwkGeneratorHs(jwa.HS256),
	jwa.HS384: JwkGeneratorHs(jwa.HS384),
	jwa.HS512: JwkGeneratorHs(jwa.HS512),
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

func JwkGeneratorEd25519() (any, any, string, string, error) {
	priv, pub, err := jwk.GenerateED25519()
	if err != nil {
		return nil, nil, "", "", err
	}

	return priv, pub, priv.KID, pub.KID, nil
}

func JwkGeneratorHs(alg jwa.Alg) func() (any, any, string, string, error) {
	return func() (any, any, string, string, error) {
		var (
			preset jwk.HMACPreset
			ok     bool
		)

		if preset, ok = JwkPresetsHMAC[alg]; !ok {
			return nil, nil, "", "", fmt.Errorf("%w (hmac): %s", ErrJwkPresetUnknown, alg)
		}

		key, err := jwk.GenerateHMAC(preset)
		if err != nil {
			return nil, nil, "", "", err
		}

		return key, nil, key.KID, "", nil
	}
}

func JwkGeneratorEs(alg jwa.Alg) func() (any, any, string, string, error) {
	return func() (any, any, string, string, error) {
		var (
			preset jwk.ECDSAPreset
			ok     bool
		)

		if preset, ok = JwkPresetsEcdsa[alg]; !ok {
			return nil, nil, "", "", fmt.Errorf("%w (es): %s", ErrJwkPresetUnknown, alg)
		}

		priv, pub, err := jwk.GenerateECDSA(preset)
		if err != nil {
			return nil, nil, "", "", err
		}

		return priv, pub, priv.KID, pub.KID, nil
	}
}

func JwkGeneratorRsa(alg jwa.Alg) func() (any, any, string, string, error) {
	return func() (any, any, string, string, error) {
		var (
			preset jwk.RSAPreset
			ok     bool
		)

		if preset, ok = JwkPresetsRsa[alg]; !ok {
			return nil, nil, "", "", fmt.Errorf("%w (rsa): %s", ErrJwkPresetUnknown, alg)
		}

		priv, pub, err := jwk.GenerateRSA(preset)
		if err != nil {
			return nil, nil, "", "", err
		}

		return priv, pub, priv.KID, pub.KID, nil
	}
}

type JwkPrivateSources struct {
	EdDSA map[string]*jwk.Source[ed25519.PrivateKey]
	HMAC  map[string]*jwk.Source[[]byte]
	ES    map[string]*jwk.Source[*ecdsa.PrivateKey]
	RSA   map[string]*jwk.Source[*rsa.PrivateKey]
}

type JwkPrivateSource interface {
	SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error)
}

func NewJwkPrivateSource(
	source JwkPrivateSource,
	keys map[string]*config.Jwk,
) (*JwkPrivateSources, error) {
	output := &JwkPrivateSources{
		EdDSA: make(map[string]*jwk.Source[ed25519.PrivateKey]),
		HMAC:  make(map[string]*jwk.Source[[]byte]),
		ES:    make(map[string]*jwk.Source[*ecdsa.PrivateKey]),
		RSA:   make(map[string]*jwk.Source[*rsa.PrivateKey]),
	}

	for usage, keyConfig := range keys {
		currentUsage := usage
		fetch := func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, currentUsage)
		}

		sourceConfig := jwk.SourceConfig{
			CacheDuration: keyConfig.Key.Cache,
			Fetch:         fetch,
		}

		switch keyConfig.Alg {
		case jwa.EdDSA:
			output.EdDSA[currentUsage] = jwk.NewED25519PrivateSource(sourceConfig)
		case jwa.HS256, jwa.HS384, jwa.HS512:
			output.HMAC[currentUsage] = jwk.NewHMACSource(sourceConfig, JwkPresetsHMAC[keyConfig.Alg])
		case jwa.ES256, jwa.ES384, jwa.ES512:
			output.ES[currentUsage] = jwk.NewECDSAPrivateSource(sourceConfig, JwkPresetsEcdsa[keyConfig.Alg])
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
			output.RSA[currentUsage] = jwk.NewRSAPrivateSource(sourceConfig, JwkPresetsRsa[keyConfig.Alg])
		default:
			return nil, fmt.Errorf("%w: %s", ErrJwkPresetUnknownAlgorithm, keyConfig.Alg)
		}
	}

	return output, nil
}

type JwkPublicSources struct {
	EdDSA map[string]*jwk.Source[ed25519.PublicKey]
	HMAC  map[string]*jwk.Source[[]byte]
	ES    map[string]*jwk.Source[*ecdsa.PublicKey]
	RSA   map[string]*jwk.Source[*rsa.PublicKey]
}

type JwkPublicSource interface {
	SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error)
}

func NewJwkPublicSource(
	source JwkPublicSource,
	keys map[string]*config.Jwk,
) (*JwkPublicSources, error) {
	output := &JwkPublicSources{
		EdDSA: make(map[string]*jwk.Source[ed25519.PublicKey]),
		HMAC:  make(map[string]*jwk.Source[[]byte]),
		ES:    make(map[string]*jwk.Source[*ecdsa.PublicKey]),
		RSA:   make(map[string]*jwk.Source[*rsa.PublicKey]),
	}

	for usage, keyConfig := range keys {
		currentUsage := usage
		fetch := func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, currentUsage)
		}
		sourceConfig := jwk.SourceConfig{
			CacheDuration: keyConfig.Key.Cache,
			Fetch:         fetch,
		}

		switch keyConfig.Alg {
		case jwa.EdDSA:
			output.EdDSA[currentUsage] = jwk.NewED25519PublicSource(sourceConfig)
		case jwa.HS256, jwa.HS384, jwa.HS512:
			output.HMAC[currentUsage] = jwk.NewHMACSource(sourceConfig, JwkPresetsHMAC[keyConfig.Alg])
		case jwa.ES256, jwa.ES384, jwa.ES512:
			output.ES[currentUsage] = jwk.NewECDSAPublicSource(sourceConfig, JwkPresetsEcdsa[keyConfig.Alg])
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
			output.RSA[currentUsage] = jwk.NewRSAPublicSource(sourceConfig, JwkPresetsRsa[keyConfig.Alg])
		default:
			return nil, fmt.Errorf("%w: %s", ErrJwkPresetUnknownAlgorithm, keyConfig.Alg)
		}
	}

	return output, nil
}

type JwkProducers map[string][]jwt.ProducerPlugin

func NewJwkProducers(
	sources *JwkPrivateSources,
	keys map[string]*config.Jwk,
) (JwkProducers, error) {
	output := make(JwkProducers)

	for usage, usageConfig := range sources.EdDSA {
		signer := jws.NewSourcedED25519Signer(usageConfig)
		output[usage] = []jwt.ProducerPlugin{signer}
	}

	for usage, usageConfig := range sources.HMAC {
		signer := jws.NewSourcedHMACSigner(usageConfig, JwsPresetsHMAC[keys[usage].Alg])
		output[usage] = append(output[usage], signer)
	}

	for usage, usageConfig := range sources.ES {
		signer := jws.NewSourcedECDSASigner(usageConfig, JwsPresetsEcdsa[keys[usage].Alg])
		output[usage] = append(output[usage], signer)
	}

	for usage, usageConfig := range sources.RSA {
		if rsaPreset, ok := JwsPresetsRsa[keys[usage].Alg]; ok {
			signer := jws.NewSourcedRSASigner(usageConfig, rsaPreset)
			output[usage] = append(output[usage], signer)
		} else if rsapssPreset, ok := JwsPresetsRsaPss[keys[usage].Alg]; ok {
			signer := jws.NewSourcedRSAPSSSigner(usageConfig, rsapssPreset)
			output[usage] = append(output[usage], signer)
		} else {
			return nil, fmt.Errorf("%w (rsa) for usage: %s", ErrJwkPresetUnknown, usage)
		}
	}

	return output, nil
}

type JwkRecipients map[string][]jwt.RecipientPlugin

func NewJwkRecipients(
	sources *JwkPublicSources,
	keys map[string]*config.Jwk,
) (JwkRecipients, error) {
	output := make(JwkRecipients)

	for usage, usageConfig := range sources.EdDSA {
		recipient := jws.NewSourcedED25519Verifier(usageConfig)
		output[usage] = []jwt.RecipientPlugin{recipient}
	}

	for usage, usageConfig := range sources.HMAC {
		recipient := jws.NewSourcedHMACVerifier(usageConfig, JwsPresetsHMAC[keys[usage].Alg])
		output[usage] = append(output[usage], recipient)
	}

	for usage, usageConfig := range sources.ES {
		recipient := jws.NewSourcedECDSAVerifier(usageConfig, JwsPresetsEcdsa[keys[usage].Alg])
		output[usage] = append(output[usage], recipient)
	}

	for usage, usageConfig := range sources.RSA {
		if rsaPreset, ok := JwsPresetsRsa[keys[usage].Alg]; ok {
			recipient := jws.NewSourcedRSAVerifier(usageConfig, rsaPreset)
			output[usage] = append(output[usage], recipient)
		} else if rsapssPreset, ok := JwsPresetsRsaPss[keys[usage].Alg]; ok {
			recipient := jws.NewSourcedRSAPSSVerifier(usageConfig, rsapssPreset)
			output[usage] = append(output[usage], recipient)
		} else {
			return nil, fmt.Errorf("%w (rsa) for usage: %s", ErrJwkPresetUnknown, usage)
		}
	}

	return output, nil
}
