package tokens

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
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
	keys map[string]*jwkconfig.Usage,
) (JwkRecipients, error) {
	output := make(JwkRecipients)

	for usage, keyConfig := range keys {
		keySource := jwk.NewSource(jwk.SourceConfig{
			Fetch: func(ctx context.Context) ([]*jwa.JWK, error) {
				return source.SearchKeys(ctx, usage)
			},
			CacheDuration:        keyConfig.Key.Cache,
			StaleIfError:         keyConfig.Key.StaleIfError,
			StaleWhileRevalidate: keyConfig.Key.StaleWhileRevalidate,
			// A rotated signing key can appear before the verifier's normal cache refresh.
			// An unknown key ID triggers one rate-limited refetch so verification can continue.
			RefreshOnUnknownKeyID: true,
			UnknownKeyIDInterval:  keyConfig.Key.UnknownKeyIDInterval,
		})

		recipient, err := newJwkVerifier(keySource, keyConfig.Alg)
		if err != nil {
			return nil, err
		}

		output[usage] = []jwt.RecipientPlugin{recipient}
	}

	return output, nil
}

// newJwkVerifier returns the plugin that verifies tokens of alg with the keys of keySource.
func newJwkVerifier(keySource *jwk.Source, alg jwa.Alg) (jwt.RecipientPlugin, error) {
	if preset, ok := jwkconfig.JwsPresetsEcdsa[alg]; ok {
		return jws.NewSourcedECDSAVerifier(keySource, preset), nil
	}

	if preset, ok := jwkconfig.JwsPresetsRsa[alg]; ok {
		return jws.NewSourcedRSAVerifier(keySource, preset), nil
	}

	switch alg {
	// The Ed25519 verifier accepts tokens and keys under either label.
	case jwa.Ed25519, jwa.EdDSA: //nolint:staticcheck // A configured legacy label is honored.
		return jws.NewSourcedED25519Verifier(keySource), nil
	case jwa.MLDSA44, jwa.MLDSA65, jwa.MLDSA87:
		return jws.NewSourcedMLDSAVerifier(keySource), nil
	default:
		return nil, fmt.Errorf("%w: %s", jwkconfig.ErrJwkPresetUnknownAlgorithm, alg)
	}
}
