package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	jwkconfig "github.com/a-novel/service-json-keys/v2/internal/config/jwk"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

// ErrJwkGenUnknownKeyUsage is returned when no key generator is registered for the requested usage's algorithm.
var ErrJwkGenUnknownKeyUsage = errors.New("unknown key usage")

// JwkGenDaoSearch is the DAO search dependency of [JwkGen].
type JwkGenDaoSearch interface {
	Exec(ctx context.Context, request *dao.JwkSearchRequest) ([]*dao.Jwk, error)
}

// JwkGenDaoInsert is the DAO insert dependency of [JwkGen].
type JwkGenDaoInsert interface {
	Exec(ctx context.Context, request *dao.JwkInsertRequest) (*dao.Jwk, error)
}

// JwkGenServiceExtract is the service dependency of [JwkGen] for deserializing generated keys.
type JwkGenServiceExtract interface {
	Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error)
}

// JwkGenRequest holds the parameters for a [JwkGen.Exec] call.
type JwkGenRequest struct {
	// Usage identifies which key configuration to use for this rotation.
	Usage string
}

// A JwkGen generates new keys for a configured usage.
//
// Generation is conditional: it reads the usage's latest key and generates only once the
// rotation window has elapsed. Within the window it returns that key.
type JwkGen struct {
	daoSearch      JwkGenDaoSearch
	daoInsert      JwkGenDaoInsert
	serviceExtract JwkGenServiceExtract
	keysConfig     map[string]*jwkconfig.Jwk
}

func NewJwkGen(
	daoSearch JwkGenDaoSearch,
	daoInsert JwkGenDaoInsert,
	serviceExtract JwkGenServiceExtract,
	keysConfig map[string]*jwkconfig.Jwk,
) *JwkGen {
	return &JwkGen{
		daoSearch:      daoSearch,
		daoInsert:      daoInsert,
		serviceExtract: serviceExtract,
		keysConfig:     keysConfig,
	}
}

func (service *JwkGen) Exec(ctx context.Context, request *JwkGenRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.JwkGen")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	// The newest key for the usage decides whether the rotation window has elapsed.
	keys, err := service.daoSearch.Exec(ctx, &dao.JwkSearchRequest{Usage: request.Usage})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list keys: %w", err))
	}

	keyConfig, ok := service.keysConfig[request.Usage]
	if !ok {
		return nil, otel.ReportError(span, ErrConfigNotFound)
	}

	latestKey := lo.FirstOrEmpty(keys)
	lastCreated := lo.FromPtr(latestKey).CreatedAt

	span.SetAttributes(attribute.Int64("key.last_created", lastCreated.Unix()))

	if time.Since(lastCreated) >= keyConfig.Key.Rotation {
		latestKey, err = service.generate(ctx, request.Usage, keyConfig)
		if err != nil {
			return nil, otel.ReportError(span, err)
		}
	}

	extractRequest := &JwkExtractRequest{
		Jwk:     latestKey,
		Private: true,
	}

	output, err := service.serviceExtract.Exec(ctx, extractRequest)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return output, nil
}

// generate creates a key for usage and stores it as the usage's newest key.
func (service *JwkGen) generate(ctx context.Context, usage string, keyConfig *jwkconfig.Jwk) (*dao.Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.JwkGen(generate)")
	defer span.End()

	keyGenerator, ok := JwkGenerators[keyConfig.Alg]
	if !ok {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", ErrJwkGenUnknownKeyUsage, usage))
	}

	generatedKey, err := keyGenerator()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate key: %w", err))
	}

	// Encrypt the private key with the master key, so a database dump does not expose it.
	privateKeyEncrypted, err := lib.EncryptMasterKey(ctx, generatedKey.PrivateKey)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt private key: %w", err))
	}

	// Both private and public keys share the same KID.
	kid, err := uuid.Parse(generatedKey.PrivateKID)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("parse KID: %w", err))
	}

	var publicKeyEncoded *string

	if generatedKey.PublicKey != nil {
		publicKeySerialized, err := json.Marshal(generatedKey.PublicKey)
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("serialize public key: %w", err))
		}

		publicKeyEncoded = lo.ToPtr(base64.RawURLEncoding.EncodeToString(publicKeySerialized))
	}

	now := time.Now()

	daoRequest := &dao.JwkInsertRequest{
		ID:         kid,
		PrivateKey: base64.RawURLEncoding.EncodeToString(privateKeyEncrypted),
		PublicKey:  publicKeyEncoded,
		Usage:      usage,
		Now:        now,
		Expiration: now.Add(keyConfig.Key.TTL),
	}

	key, err := service.daoInsert.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("insert key: %w", err))
	}

	return key, nil
}
