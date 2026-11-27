package tokens

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"
	"github.com/a-novel-kit/jwt/v2/jwp"
	"github.com/a-novel-kit/jwt/v2/jws"
)

const (
	// APIKeyCacheDuration is how long a verifier keeps an API key it read. A key deactivated since
	// authenticates requests for as long.
	APIKeyCacheDuration = 5 * time.Minute
	// APIKeyLeeway is the clock skew tolerated between the owner of an API key and a verifier.
	APIKeyLeeway = 30 * time.Second
	// APIKeyCacheSize bounds the keys a verifier keeps. A caller picks the keys its tokens name, so
	// a cache that reaches it starts over.
	APIKeyCacheSize = 1024
)

var (
	// ErrAPIKeyNotFound is returned by an [APIKeySource] that holds no key with the requested ID.
	ErrAPIKeyNotFound = errors.New("api key not found")
	// ErrAPIKeySource is returned when an API key could not be read, or is not one a verifier can
	// use. The token it would have verified is not at fault.
	ErrAPIKeySource = errors.New("read api key")
	// errSignedAt reports a token signed at no acceptable time, or one that lives too long after it.
	errSignedAt = errors.New("invalid issue date")
	// ErrPurposeRequired is returned by [NewAPIKeys] without a purpose, which would let a key made
	// for anything verify.
	ErrPurposeRequired = errors.New("api key purpose required")
)

// APIKey is an API key as the service serves it: a public key made for one purpose, with its
// states.
type APIKey struct {
	// ID names the key. A token the key signed carries it in its "kid" header.
	ID string
	// Purpose is the one thing the key may do.
	Purpose string
	// Key is the public key.
	Key *jwa.JWK
	// Deactivated is whether the key no longer authenticates a request. What it signed before still
	// verifies.
	Deactivated bool
	// Leaked is whether the private key leaked, so nothing the key signed is trusted.
	Leaked bool
}

// APIKeySource reads the API keys an [APIKeys] verifies with.
type APIKeySource interface {
	// SelectKey returns the API key of id, or [ErrAPIKeyNotFound].
	SelectKey(ctx context.Context, id string) (*APIKey, error)
}

// An APIKeys verifies tokens with the API keys of one purpose. It reads the key a token names from
// its source, and keeps it for [APIKeyCacheDuration]. Build one with [NewAPIKeys].
type APIKeys struct {
	source  APIKeySource
	purpose string

	lock  sync.Mutex
	cache map[string]*cachedAPIKey
}

// A cachedAPIKey is an API key a verifier may use, with the plugin that verifies its signatures.
type cachedAPIKey struct {
	key      *APIKey
	verifier jwt.RecipientPlugin
	expires  time.Time
}

// An apiKeyToken is one kind of token an API key signs.
type apiKeyToken struct {
	// typ is the media type the header of the token declares. A verifier checks it exactly, so one
	// kind never passes for another (RFC 8725 §3.11).
	typ jwa.Typ
	// deactivated is whether a key deactivated since still verifies the token.
	deactivated bool
	// checker checks the claims of the token before they are decoded.
	checker *jwp.ClaimsChecker
}

// NewAPIKeys returns an APIKeys for the keys source holds for purpose.
func NewAPIKeys(source APIKeySource, purpose string) (*APIKeys, error) {
	if purpose == "" {
		return nil, ErrPurposeRequired
	}

	return &APIKeys{source: source, purpose: purpose, cache: make(map[string]*cachedAPIKey)}, nil
}

// newToken returns a kind of token the keys sign for their purpose. A positive lifetime makes the
// token expire, that long after it was signed at most.
func (keys *APIKeys) newToken(typ jwa.Typ, deactivated bool, lifetime time.Duration) apiKeyToken {
	checker := jwp.NewClaimsChecker(&jwp.ClaimsCheckerConfig{Checks: []jwp.ClaimsCheck{
		jwp.NewClaimsCheckTarget(jwt.TargetConfig{Audience: jwa.Audience{keys.purpose}}),
		jwp.NewClaimsCheckTimestamp(APIKeyLeeway, lifetime > 0),
		apiKeyIssueCheck{lifetime: lifetime},
	}})

	return apiKeyToken{typ: typ, deactivated: deactivated, checker: checker}
}

// consume verifies token as one of kind, with the API key its header names, and decodes its claims
// into dst. It returns that key.
//
// A token the key does not vouch for fails with an error matching [jwt.ErrTokenRejected], and a key
// that could not be read with one matching [ErrAPIKeySource].
func (keys *APIKeys) consume(ctx context.Context, kind apiKeyToken, token string, dst any) (*APIKey, error) {
	plugin := &apiKeyPlugin{keys: keys, deactivated: kind.deactivated}
	recipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins:      []jwt.RecipientPlugin{plugin},
		Deserializer: kind.checker.Unmarshal,
		Typ:          kind.typ,
	})

	err := recipient.Consume(ctx, token, dst)
	if errors.Is(err, ErrAPIKeySource) {
		return nil, err
	}

	if err != nil {
		// Every other failure is the token's own, claims that fail to decode included.
		return nil, fmt.Errorf("%w: %w", jwt.ErrTokenRejected, err)
	}

	return plugin.key, nil
}

// get returns the key of id, from the cache or from the source once the cached one is too old. The
// key is one a verifier may use: made for the purpose, and not leaked.
func (keys *APIKeys) get(ctx context.Context, id string) (*cachedAPIKey, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: the token names no api key", jwt.ErrTokenRejected)
	}

	now := time.Now()

	keys.lock.Lock()
	cached, found := keys.cache[id]
	keys.lock.Unlock()

	if found && now.Before(cached.expires) {
		return cached, nil
	}

	// The source is read outside the lock, so two requests that miss together both read it.
	key, err := keys.source.SelectKey(ctx, id)
	if errors.Is(err, ErrAPIKeyNotFound) {
		return nil, fmt.Errorf("%w: unknown api key", jwt.ErrTokenRejected)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAPIKeySource, err)
	}

	cached, err = keys.accept(key, id)
	if err != nil {
		return nil, err
	}

	cached.expires = now.Add(APIKeyCacheDuration)

	keys.lock.Lock()
	defer keys.lock.Unlock()

	if len(keys.cache) >= APIKeyCacheSize {
		clear(keys.cache)
	}

	keys.cache[id] = cached

	return cached, nil
}

// accept returns key ready to verify with, once it is the key of id a verifier may use.
func (keys *APIKeys) accept(key *APIKey, id string) (*cachedAPIKey, error) {
	if key.ID != id {
		return nil, fmt.Errorf("%w: the source answered with another key", ErrAPIKeySource)
	}

	if key.Purpose != keys.purpose {
		return nil, fmt.Errorf("%w: the api key is made for another purpose", jwt.ErrTokenRejected)
	}

	if key.Leaked {
		return nil, fmt.Errorf("%w: the api key leaked", jwt.ErrTokenRejected)
	}

	privateKey, publicKey, err := jwk.ConsumeED25519(key.Key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAPIKeySource, err)
	}

	if privateKey != nil {
		return nil, fmt.Errorf("%w: the source exposes a private key", ErrAPIKeySource)
	}

	return &cachedAPIKey{key: key, verifier: jws.NewED25519Verifier(publicKey.Key())}, nil
}

// An apiKeyPlugin is the recipient plugin of one verification. It verifies the token with the API
// key its header names, and keeps that key for the caller.
type apiKeyPlugin struct {
	keys        *APIKeys
	deactivated bool

	key *APIKey
}

func (plugin *apiKeyPlugin) Transform(ctx context.Context, header *jwa.JWH, token string) ([]byte, error) {
	cached, err := plugin.keys.get(ctx, header.KID)
	if err != nil {
		return nil, err
	}

	if cached.key.Deactivated && !plugin.deactivated {
		return nil, fmt.Errorf("%w: the api key is deactivated", jwt.ErrTokenRejected)
	}

	payload, err := cached.verifier.Transform(ctx, header, token)
	if err != nil {
		return nil, err
	}

	plugin.key = cached.key

	return payload, nil
}

// An apiKeyIssueCheck checks when a token was signed: its "iat" claim is set, and no later than now
// within [APIKeyLeeway]. With a lifetime, the token also expires that long after "iat" at most.
type apiKeyIssueCheck struct {
	lifetime time.Duration
}

func (check apiKeyIssueCheck) CheckClaims(claims *jwa.Claims) error {
	signedAt := time.Unix(claims.Iat, 0)

	switch {
	case claims.Iat <= 0:
		return fmt.Errorf("%w: missing", errSignedAt)
	case signedAt.After(time.Now().Add(APIKeyLeeway)):
		return fmt.Errorf("%w: signed in the future, at %s", errSignedAt, signedAt)
	case check.lifetime <= 0:
		return nil
	}

	lifetime := time.Unix(claims.Exp, 0).Sub(signedAt)
	if lifetime <= 0 || lifetime > check.lifetime {
		return fmt.Errorf("%w: the token lives %s, expected %s at most", errSignedAt, lifetime, check.lifetime)
	}

	return nil
}
