package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/models"
)

var HMACPresets = map[jwa.Alg]jwk.HMACPreset{
	jwa.HS256: jwk.HS256,
	jwa.HS384: jwk.HS384,
	jwa.HS512: jwk.HS512,
}

var HMACPresetsJWS = map[jwa.Alg]jws.HMACPreset{
	jwa.HS256: jws.HS256,
	jwa.HS384: jws.HS384,
	jwa.HS512: jws.HS512,
}

var ECDSAPresets = map[jwa.Alg]jwk.ECDSAPreset{
	jwa.ES256: jwk.ES256,
	jwa.ES384: jwk.ES384,
	jwa.ES512: jwk.ES512,
}

var ECDSAPresetsJWS = map[jwa.Alg]jws.ECDSAPreset{
	jwa.ES256: jws.ES256,
	jwa.ES384: jws.ES384,
	jwa.ES512: jws.ES512,
}

var RSAPresets = map[jwa.Alg]jwk.RSAPreset{
	jwa.RS256: jwk.RS256,
	jwa.RS384: jwk.RS384,
	jwa.RS512: jwk.RS512,
	jwa.PS256: jwk.PS256,
	jwa.PS384: jwk.PS384,
	jwa.PS512: jwk.PS512,
}

var RSAPresetsJWS = map[jwa.Alg]jws.RSAPreset{
	jwa.RS256: jws.RS256,
	jwa.RS384: jws.RS384,
	jwa.RS512: jws.RS512,
}

var RSAPSSPresetsJWS = map[jwa.Alg]jws.RSAPSSPreset{
	jwa.PS256: jws.PS256,
	jwa.PS384: jws.PS384,
	jwa.PS512: jws.PS512,
}

type AnyKeyGen func() (any, any, string, string, error)

var KeyGenerators = map[jwa.Alg]AnyKeyGen{
	jwa.EdDSA: Ed25519KeyGen,
	jwa.HS256: HSKeyGen(jwa.HS256),
	jwa.HS384: HSKeyGen(jwa.HS384),
	jwa.HS512: HSKeyGen(jwa.HS512),
	jwa.ES256: ESKeyGen(jwa.ES256),
	jwa.ES384: ESKeyGen(jwa.ES384),
	jwa.ES512: ESKeyGen(jwa.ES512),
	jwa.RS256: RSAKeyGen(jwa.RS256),
	jwa.RS384: RSAKeyGen(jwa.RS384),
	jwa.RS512: RSAKeyGen(jwa.RS512),
	jwa.PS256: RSAKeyGen(jwa.PS256),
	jwa.PS384: RSAKeyGen(jwa.PS384),
	jwa.PS512: RSAKeyGen(jwa.PS512),
}

func Ed25519KeyGen() (any, any, string, string, error) {
	priv, pub, err := jwk.GenerateED25519()
	if err != nil {
		return nil, nil, "", "", err
	}

	return priv, pub, priv.KID, pub.KID, nil
}

func HSKeyGen(alg jwa.Alg) func() (any, any, string, string, error) {
	return func() (any, any, string, string, error) {
		var (
			preset jwk.HMACPreset
			ok     bool
		)

		if preset, ok = HMACPresets[alg]; !ok {
			return nil, nil, "", "", fmt.Errorf("unknown HMAC preset for algorithm: %s", alg)
		}

		key, err := jwk.GenerateHMAC(preset)
		if err != nil {
			return nil, nil, "", "", err
		}

		return key, nil, key.KID, "", nil
	}
}

func ESKeyGen(alg jwa.Alg) func() (any, any, string, string, error) {
	return func() (any, any, string, string, error) {
		var (
			preset jwk.ECDSAPreset
			ok     bool
		)

		if preset, ok = ECDSAPresets[alg]; !ok {
			return nil, nil, "", "", fmt.Errorf("unknown ECDSA preset for algorithm: %s", alg)
		}

		priv, pub, err := jwk.GenerateECDSA(preset)
		if err != nil {
			return nil, nil, "", "", err
		}

		return priv, pub, priv.KID, pub.KID, nil
	}
}

func RSAKeyGen(alg jwa.Alg) func() (any, any, string, string, error) {
	return func() (any, any, string, string, error) {
		var (
			preset jwk.RSAPreset
			ok     bool
		)

		if preset, ok = RSAPresets[alg]; !ok {
			return nil, nil, "", "", fmt.Errorf("unknown RSA preset for algorithm: %s", alg)
		}

		priv, pub, err := jwk.GenerateRSA(preset)
		if err != nil {
			return nil, nil, "", "", err
		}

		return priv, pub, priv.KID, pub.KID, nil
	}
}

type PrivateKeysSourceType struct {
	EdDSA map[models.KeyUsage]*jwk.Source[ed25519.PrivateKey]
	HMAC  map[models.KeyUsage]*jwk.Source[[]byte]
	ES    map[models.KeyUsage]*jwk.Source[*ecdsa.PrivateKey]
	RSA   map[models.KeyUsage]*jwk.Source[*rsa.PrivateKey]
}

type PrivateKeyGenericSource interface {
	SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*jwa.JWK, error)
}

func NewPrivateKeySources(source PrivateKeyGenericSource) (*PrivateKeysSourceType, error) {
	output := &PrivateKeysSourceType{
		EdDSA: make(map[models.KeyUsage]*jwk.Source[ed25519.PrivateKey]),
		HMAC:  make(map[models.KeyUsage]*jwk.Source[[]byte]),
		ES:    make(map[models.KeyUsage]*jwk.Source[*ecdsa.PrivateKey]),
		RSA:   make(map[models.KeyUsage]*jwk.Source[*rsa.PrivateKey]),
	}

	for usage, keyConfig := range config.Keys {
		fetch := func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, usage)
		}
		sourceConfig := jwk.SourceConfig{
			CacheDuration: keyConfig.Key.Cache,
			Fetch:         fetch,
		}

		switch keyConfig.Alg {
		case jwa.EdDSA:
			output.EdDSA[usage] = jwk.NewED25519PrivateSource(sourceConfig)
		case jwa.HS256, jwa.HS384, jwa.HS512:
			output.HMAC[usage] = jwk.NewHMACSource(sourceConfig, HMACPresets[keyConfig.Alg])
		case jwa.ES256, jwa.ES384, jwa.ES512:
			output.ES[usage] = jwk.NewECDSAPrivateSource(sourceConfig, ECDSAPresets[keyConfig.Alg])
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
			output.RSA[usage] = jwk.NewRSAPrivateSource(sourceConfig, RSAPresets[keyConfig.Alg])
		default:
			return nil, fmt.Errorf("unknown algorithm: %s", keyConfig.Alg)
		}
	}

	return output, nil
}

type PublicKeySourceType struct {
	EdDSA map[models.KeyUsage]*jwk.Source[ed25519.PublicKey]
	HMAC  map[models.KeyUsage]*jwk.Source[[]byte]
	ES    map[models.KeyUsage]*jwk.Source[*ecdsa.PublicKey]
	RSA   map[models.KeyUsage]*jwk.Source[*rsa.PublicKey]
}

type PublicKeyGenericSource interface {
	SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*jwa.JWK, error)
}

func NewPublicKeySource(source PublicKeyGenericSource) (*PublicKeySourceType, error) {
	output := &PublicKeySourceType{
		EdDSA: make(map[models.KeyUsage]*jwk.Source[ed25519.PublicKey]),
		HMAC:  make(map[models.KeyUsage]*jwk.Source[[]byte]),
		ES:    make(map[models.KeyUsage]*jwk.Source[*ecdsa.PublicKey]),
		RSA:   make(map[models.KeyUsage]*jwk.Source[*rsa.PublicKey]),
	}

	for usage, keyConfig := range config.Keys {
		fetch := func(ctx context.Context) ([]*jwa.JWK, error) {
			return source.SearchKeys(ctx, usage)
		}
		sourceConfig := jwk.SourceConfig{
			CacheDuration: keyConfig.Key.Cache,
			Fetch:         fetch,
		}

		switch keyConfig.Alg {
		case jwa.EdDSA:
			output.EdDSA[usage] = jwk.NewED25519PublicSource(sourceConfig)
		case jwa.HS256, jwa.HS384, jwa.HS512:
			output.HMAC[usage] = jwk.NewHMACSource(sourceConfig, HMACPresets[keyConfig.Alg])
		case jwa.ES256, jwa.ES384, jwa.ES512:
			output.ES[usage] = jwk.NewECDSAPublicSource(sourceConfig, ECDSAPresets[keyConfig.Alg])
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
			output.RSA[usage] = jwk.NewRSAPublicSource(sourceConfig, RSAPresets[keyConfig.Alg])
		default:
			return nil, fmt.Errorf("unknown algorithm: %s", keyConfig.Alg)
		}
	}

	return output, nil
}

type Producers map[models.KeyUsage][]jwt.ProducerPlugin

func NewProducers(sources *PrivateKeysSourceType) (Producers, error) {
	output := make(Producers)

	for usage, usageConfig := range sources.EdDSA {
		signer := jws.NewSourcedED25519Signer(usageConfig)
		output[usage] = []jwt.ProducerPlugin{signer}
	}

	for usage, usageConfig := range sources.HMAC {
		signer := jws.NewSourcedHMACSigner(usageConfig, HMACPresetsJWS[config.Keys[usage].Alg])
		output[usage] = append(output[usage], signer)
	}

	for usage, usageConfig := range sources.ES {
		signer := jws.NewSourcedECDSASigner(usageConfig, ECDSAPresetsJWS[config.Keys[usage].Alg])
		output[usage] = append(output[usage], signer)
	}

	for usage, usageConfig := range sources.RSA {
		if rsaPreset, ok := RSAPresetsJWS[config.Keys[usage].Alg]; ok {
			signer := jws.NewSourcedRSASigner(usageConfig, rsaPreset)
			output[usage] = append(output[usage], signer)
		} else if rsapssPreset, ok := RSAPSSPresetsJWS[config.Keys[usage].Alg]; ok {
			signer := jws.NewSourcedRSAPSSSigner(usageConfig, rsapssPreset)
			output[usage] = append(output[usage], signer)
		} else {
			return nil, fmt.Errorf("unknown RSA preset for usage: %s", usage)
		}
	}

	return output, nil
}

type Recipients map[models.KeyUsage][]jwt.RecipientPlugin

func NewRecipients(sources *PublicKeySourceType) (Recipients, error) {
	output := make(Recipients)

	for usage, usageConfig := range sources.EdDSA {
		recipient := jws.NewSourcedED25519Verifier(usageConfig)
		output[usage] = []jwt.RecipientPlugin{recipient}
	}

	for usage, usageConfig := range sources.HMAC {
		recipient := jws.NewSourcedHMACVerifier(usageConfig, HMACPresetsJWS[config.Keys[usage].Alg])
		output[usage] = append(output[usage], recipient)
	}

	for usage, usageConfig := range sources.ES {
		recipient := jws.NewSourcedECDSAVerifier(usageConfig, ECDSAPresetsJWS[config.Keys[usage].Alg])
		output[usage] = append(output[usage], recipient)
	}

	for usage, usageConfig := range sources.RSA {
		if rsaPreset, ok := RSAPresetsJWS[config.Keys[usage].Alg]; ok {
			recipient := jws.NewSourcedRSAVerifier(usageConfig, rsaPreset)
			output[usage] = append(output[usage], recipient)
		} else if rsapssPreset, ok := RSAPSSPresetsJWS[config.Keys[usage].Alg]; ok {
			recipient := jws.NewSourcedRSAPSSVerifier(usageConfig, rsapssPreset)
			output[usage] = append(output[usage], recipient)
		} else {
			return nil, fmt.Errorf("unknown RSA preset for usage: %s", usage)
		}
	}

	return output, nil
}
