package core

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"maps"
	"slices"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jwk/serializers"
)

// pemPublicKey is the type of the PEM block that holds a public key (RFC 7468 §13).
const pemPublicKey = "PUBLIC KEY"

// pemBegin opens the first line of every PEM block (RFC 7468 §2).
var pemBegin = []byte("-----BEGIN ")

// APIKeySelectRequest holds the parameters for an [APIKeySelect.Exec] call.
type APIKeySelectRequest struct {
	// ID is the API key to retrieve; it corresponds to the "kid" header of a token the key signed.
	ID string
}

// An APIKeySelect retrieves the API key matching an ID, among the system API keys the service reads
// from its settings.
type APIKeySelect struct {
	keys map[string]*APIKey
}

// NewAPIKeySelect reads the system API keys of each purpose from its setting: one or more Ed25519
// public keys in PEM, the active one first. Every other key of a setting is deactivated. The ID of a
// key is the JWK thumbprint of its public key (RFC 7638).
//
// A setting that holds anything else, or a key listed twice, returns [ErrInvalidAPIKeys]. The error
// names the purpose and the position of the key, never its content.
func NewAPIKeySelect(settings map[string]string) (*APIKeySelect, error) {
	keys := make(map[string]*APIKey)

	// Go randomizes the order of a map, and the first invalid setting is the one reported.
	for _, purpose := range slices.Sorted(maps.Keys(settings)) {
		publicKeys, err := parsePublicKeys(settings[purpose])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", purpose, err)
		}

		for position, publicKey := range publicKeys {
			key, err := newSystemAPIKey(purpose, publicKey, position > 0)
			if err != nil {
				return nil, fmt.Errorf("%s: %w: key %d: %w", purpose, ErrInvalidAPIKeys, position+1, err)
			}

			if other, found := keys[key.ID]; found {
				return nil, fmt.Errorf(
					"%s: %w: key %d is already a key of %s", purpose, ErrInvalidAPIKeys, position+1, other.Purpose,
				)
			}

			keys[key.ID] = key
		}
	}

	return &APIKeySelect{keys: keys}, nil
}

// Exec returns the API key of the request's ID, which every caller shares and none may change. It
// returns [ErrAPIKeyNotFound] when the service holds no such key.
func (service *APIKeySelect) Exec(ctx context.Context, request *APIKeySelectRequest) (*APIKey, error) {
	_, span := otel.Tracer().Start(ctx, "core.APIKeySelect")
	defer span.End()

	span.SetAttributes(attribute.String("key.id", request.ID))

	key, found := service.keys[request.ID]
	if !found {
		return nil, otel.ReportError(span, ErrAPIKeyNotFound)
	}

	return key, nil
}

// parsePublicKeys reads the Ed25519 public keys of a setting, in order. An empty setting holds none.
// No error quotes the setting: a private key pasted into it by mistake must not reach a log.
func parsePublicKeys(setting string) ([]ed25519.PublicKey, error) {
	var keys []ed25519.PublicKey

	rest := bytes.TrimSpace([]byte(setting))

	for len(rest) > 0 {
		// pem.Decode skips any text before a block, a malformed block included, so the block must open
		// the rest and be the only one read.
		block, next := pem.Decode(rest)
		read := rest[:len(rest)-len(next)]
		rest = next

		if block == nil || !bytes.HasPrefix(read, pemBegin) || bytes.Count(read, pemBegin) > 1 {
			return nil, fmt.Errorf("%w: key %d: not a PEM block", ErrInvalidAPIKeys, len(keys)+1)
		}

		if block.Type != pemPublicKey {
			return nil, fmt.Errorf("%w: key %d: not a %s block", ErrInvalidAPIKeys, len(keys)+1, pemPublicKey)
		}

		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: key %d: not a public key", ErrInvalidAPIKeys, len(keys)+1)
		}

		key, isEd25519 := parsed.(ed25519.PublicKey)
		if !isEd25519 {
			return nil, fmt.Errorf("%w: key %d: not an Ed25519 key", ErrInvalidAPIKeys, len(keys)+1)
		}

		keys = append(keys, key)
		rest = bytes.TrimSpace(rest)
	}

	return keys, nil
}

// newSystemAPIKey returns the API key of a public key the settings hold for a purpose.
func newSystemAPIKey(purpose string, publicKey ed25519.PublicKey, deactivated bool) (*APIKey, error) {
	payload, err := json.Marshal(serializers.EncodeED(publicKey))
	if err != nil {
		return nil, fmt.Errorf("encode key: %w", err)
	}

	key := &Jwk{
		KTY:     jwa.KTYOKP,
		Use:     jwa.UseSig,
		KeyOps:  jwa.KeyOps{jwa.KeyOpVerify},
		Alg:     jwa.Ed25519,
		Payload: payload,
	}

	key.KID, err = jwk.Thumbprint(key)
	if err != nil {
		return nil, fmt.Errorf("compute thumbprint: %w", err)
	}

	return &APIKey{ID: key.KID, Purpose: purpose, Key: key, Deactivated: deactivated}, nil
}
