package servicejsonkeystest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jws"

	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

const (
	// TypRequestToken is the type of a request token, which authenticates one request.
	TypRequestToken = "JWT"
	// TypProof is the type of a proof, which stays with the entry a request creates.
	TypProof = "proof+jwt"
)

// An APIKey is an API key made for a test. Its private key lives in memory only, so what it signs
// dies with the process.
type APIKey struct {
	// ID is the ID of the key: the JWK thumbprint of its public key, as a system API key has.
	ID string
	// Purpose is the purpose the key is made for.
	Purpose string
	// Deactivated and Leaked are the states the key is served in. A test sets them before the key
	// is first read.
	Deactivated bool
	Leaked      bool
	// PublicKey is the public key, as the service serves it.
	PublicKey *jwa.JWK

	privateKey *jwa.JWK
}

// NewAPIKey generates an API key made for purpose.
func NewAPIKey(t *testing.T, purpose string) *APIKey {
	t.Helper()

	privateKey, publicKey, err := jwk.GenerateED25519()
	if err != nil {
		panic(err)
	}

	id, err := jwk.Thumbprint(publicKey.JWK)
	if err != nil {
		panic(err)
	}

	publicKey.KID = id

	return &APIKey{ID: id, Purpose: purpose, PublicKey: publicKey.JWK, privateKey: privateKey.JWK}
}

// RequestClaims returns the claims of a request token signed now for body, which lives as long as
// one may. A test changes them to build a token a verifier refuses.
func (key *APIKey) RequestClaims(body []byte) map[string]any {
	now := time.Now()
	digest := sha256.Sum256(body)

	return map[string]any{
		"aud":        key.Purpose,
		"iat":        now.Unix(),
		"exp":        now.Add(servicejsonkeys.APIKeyRequestLifetime).Unix(),
		"bodySHA256": hex.EncodeToString(digest[:]),
	}
}

// ProofClaims returns the claims of a proof signed now: entry, which says what the proof vouches
// for, beside the purpose of the key.
func (key *APIKey) ProofClaims(entry map[string]any) map[string]any {
	claims := map[string]any{"aud": key.Purpose, "iat": time.Now().Unix()}
	maps.Copy(claims, entry)

	return claims
}

// Sign returns claims as a token of type typ signed with the key, whose header names the key.
// Claims are those of [APIKey.RequestClaims] or [APIKey.ProofClaims], or any value that encodes as
// the claims a test needs.
func (key *APIKey) Sign(t *testing.T, typ string, claims any) string {
	t.Helper()

	// The signer stamps the ID of the key it draws from its source into the header.
	privateKey := *key.privateKey
	privateKey.KID = key.ID

	source := jwk.NewSource(jwk.SourceConfig{
		Fetch: func(context.Context) ([]*jwa.JWK, error) { return []*jwa.JWK{&privateKey}, nil },
	})

	producer := jwt.NewProducer(jwt.ProducerConfig{
		Header: jwt.HeaderProducerConfig{Typ: jwa.Typ(typ)},
		// A token an API key signs is labeled "EdDSA", the name its format gives the algorithm.
		Plugins: []jwt.ProducerPlugin{jws.NewSourcedEdDSASigner(source)}, //nolint:staticcheck // See above.
	})

	token, err := producer.Issue(t.Context(), claims, nil)
	if err != nil {
		panic(err)
	}

	return token
}

// APIKeys serves API keys made for a test, as the service serves its own. It is a
// [servicejsonkeys.APIKeyReader].
type APIKeys []*APIKey

// APIKeyGet returns the key of the requested ID, or a NotFound status.
func (keys APIKeys) APIKeyGet(
	_ context.Context, req *servicejsonkeys.APIKeyGetRequest, _ ...grpc.CallOption,
) (*servicejsonkeys.APIKeyGetResponse, error) {
	for _, key := range keys {
		if key.ID != req.GetId() {
			continue
		}

		return &servicejsonkeys.APIKeyGetResponse{ApiKey: &jsonkeysv2.APIKey{
			Id:      key.ID,
			Purpose: key.Purpose,
			Jwk: &jsonkeysv2.Jwk{
				Kty:     key.PublicKey.KTY.String(),
				Use:     key.PublicKey.Use.String(),
				KeyOps:  key.PublicKey.KeyOps.Strings(),
				Alg:     key.PublicKey.Alg.String(),
				Kid:     key.PublicKey.KID,
				Payload: key.PublicKey.Payload,
			},
			Deactivated: key.Deactivated,
			Leaked:      key.Leaked,
		}}, nil
	}

	return nil, status.Error(codes.NotFound, "api key not found")
}

// WithAPIKeys returns client reading its API keys from keys instead of the service. Every other
// call goes to client.
func WithAPIKeys(client servicejsonkeys.Client, keys ...*APIKey) servicejsonkeys.Client {
	return &apiKeysClient{Client: client, keys: keys}
}

type apiKeysClient struct {
	servicejsonkeys.Client

	keys APIKeys
}

func (client *apiKeysClient) APIKeyGet(
	ctx context.Context, req *servicejsonkeys.APIKeyGetRequest, opts ...grpc.CallOption,
) (*servicejsonkeys.APIKeyGetResponse, error) {
	return client.keys.APIKeyGet(ctx, req, opts...)
}
