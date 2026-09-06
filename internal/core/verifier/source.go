package verifier

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	jwtjwk "github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
)

// Source provides the public JWKs used by NewRecipients.
type Source interface {
	// SearchKeys returns the public keys registered for usage.
	SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error)
}

func newSource(
	searchKeys func(context.Context, string) ([]*jwa.JWK, error),
	usage string,
	sourceConfig jwtjwk.SourceConfig,
) *jwtjwk.Source {
	sourceConfig.Fetch = func(ctx context.Context) ([]*jwa.JWK, error) {
		return searchKeys(ctx, usage)
	}

	return jwtjwk.NewSource(sourceConfig)
}

// Recipients maps each usage to the JWT verification plugins for its algorithm.
type Recipients map[string][]jwt.RecipientPlugin

// NewRecipients builds cached verification plugins from public keys for every configured usage.
func NewRecipients(source Source, keys map[string]*jwkconfig.Jwk) (Recipients, error) {
	output := make(Recipients)

	for usage, keyConfig := range keys {
		keySource := newSource(source.SearchKeys, usage, jwtjwk.SourceConfig{
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
			preset, ok := jwkconfig.JwsPresetsEcdsa[keyConfig.Alg]
			if !ok {
				return nil, fmt.Errorf("%w (ecdsa) for usage: %s", ErrPresetUnknown, usage)
			}

			recipient = jws.NewSourcedECDSAVerifier(keySource, preset)
		case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
			preset, ok := jwkconfig.JwsPresetsRsa[keyConfig.Alg]
			if !ok {
				return nil, fmt.Errorf("%w (rsa) for usage: %s", ErrPresetUnknown, usage)
			}

			recipient = jws.NewSourcedRSAVerifier(keySource, preset)
		default:
			return nil, fmt.Errorf("%w: %s", ErrPresetUnknownAlgorithm, keyConfig.Alg)
		}

		output[usage] = []jwt.RecipientPlugin{recipient}
	}

	return output, nil
}
