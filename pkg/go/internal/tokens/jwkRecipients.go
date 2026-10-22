package tokens

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
)

// JwkPublicSource provides the public JWKs used by [NewJwkRecipients].
type JwkPublicSource interface {
	// SearchKeys returns the public JWKs registered for usage.
	SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error)
}

// JwkRecipients maps each key usage to the set of JWT recipient plugins used for verifying tokens
// under that usage.
type JwkRecipients map[string][]jwt.RecipientPlugin

// NewJwkRecipients builds cached verification plugins from public keys for every configured usage.
func NewJwkRecipients(
	source JwkPublicSource,
	keys map[string]*jwkconfig.Jwk,
) (JwkRecipients, error) {
	output := make(JwkRecipients)

	for usage, keyConfig := range keys {
		sourceConfig := jwk.SourceConfig{
			CacheDuration:        keyConfig.Key.Cache,
			StaleIfError:         keyConfig.Key.StaleIfError,
			StaleWhileRevalidate: keyConfig.Key.StaleWhileRevalidate,
			// A rotated signing key can appear before the verifier's normal cache refresh.
			// An unknown key ID triggers one rate-limited refetch so verification can continue.
			RefreshOnUnknownKeyID: true,
			UnknownKeyIDInterval:  keyConfig.Key.UnknownKeyIDInterval,
		}

		keySource := newJwkSource(source.SearchKeys, usage, sourceConfig)

		recipient, err := newJwkVerifier(keySource, usage, keyConfig.Alg)
		if err != nil {
			return nil, err
		}

		output[usage] = []jwt.RecipientPlugin{recipient}
	}

	return output, nil
}

// newJwkSource returns a key source that caches the keys searchKeys returns for usage.
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

// newJwkVerifier returns the plugin that verifies the tokens of usage with the keys of keySource.
func newJwkVerifier(keySource *jwk.Source, usage string, alg jwa.Alg) (jwt.RecipientPlugin, error) {
	switch alg {
	// The Ed25519 verifier accepts tokens and keys under either label.
	case jwa.Ed25519, jwa.EdDSA: //nolint:staticcheck // A configured legacy label is honored.
		return jws.NewSourcedED25519Verifier(keySource), nil
	case jwa.ES256, jwa.ES384, jwa.ES512:
		ecdsaPreset, ok := jwkconfig.JwsPresetsEcdsa[alg]
		if !ok {
			return nil, fmt.Errorf("%w (ecdsa) for usage: %s", jwkconfig.ErrJwkPresetUnknown, usage)
		}

		return jws.NewSourcedECDSAVerifier(keySource, ecdsaPreset), nil
	case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
		rsaPreset, ok := jwkconfig.JwsPresetsRsa[alg]
		if !ok {
			return nil, fmt.Errorf("%w (rsa) for usage: %s", jwkconfig.ErrJwkPresetUnknown, usage)
		}

		return jws.NewSourcedRSAVerifier(keySource, rsaPreset), nil
	case jwa.MLDSA44, jwa.MLDSA65, jwa.MLDSA87:
		return jws.NewSourcedMLDSAVerifier(keySource), nil
	default:
		return nil, fmt.Errorf("%w: %s", jwkconfig.ErrJwkPresetUnknownAlgorithm, alg)
	}
}
