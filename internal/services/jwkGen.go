package services

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
	"go.opentelemetry.io/otel/trace"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

// ErrJwkGenUnknownKeyUsage is returned when no key generator is registered for the requested usage's algorithm.
var ErrJwkGenUnknownKeyUsage = errors.New("unknown key usage")

// JwkGenRepositorySearch is the DAO search dependency of [JwkGen].
type JwkGenRepositorySearch interface {
	Exec(ctx context.Context, request *dao.JwkSearchRequest) ([]*dao.Jwk, error)
}

// JwkGenRepositoryInsert is the DAO insert dependency of [JwkGen].
type JwkGenRepositoryInsert interface {
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
// It does not force the generation of a key. Instead, when called, it checks the
// current state of the database to determine whether a new main key is needed.
//
// If the main key for the target usage is recent enough, this service returns it
// and skips generation. This is recorded in traces.
type JwkGen struct {
	repositorySearch JwkGenRepositorySearch
	repositoryInsert JwkGenRepositoryInsert
	serviceExtract   JwkGenServiceExtract
	keysConfig       map[string]*config.Jwk
}

// NewJwkGen returns a new JwkGen service.
func NewJwkGen(
	repositorySearch JwkGenRepositorySearch,
	repositoryInsert JwkGenRepositoryInsert,
	serviceExtract JwkGenServiceExtract,
	keysConfig map[string]*config.Jwk,
) *JwkGen {
	return &JwkGen{
		repositorySearch: repositorySearch,
		repositoryInsert: repositoryInsert,
		serviceExtract:   serviceExtract,
		keysConfig:       keysConfig,
	}
}

func (service *JwkGen) Exec(ctx context.Context, request *JwkGenRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "services.JwkGen")
	defer span.End()

	span.SetAttributes(attribute.String("key.usage", request.Usage))

	// Check the last time a key was inserted for the target usage, and compare to config. If the last key is too
	// recent, return without generating a new key.
	keys, err := service.repositorySearch.Exec(ctx, &dao.JwkSearchRequest{Usage: request.Usage})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list keys: %w", err))
	}

	span.AddEvent("keys.retrieved", trace.WithAttributes(attribute.Int("keys.count", len(keys))))

	var lastCreated time.Time
	if len(keys) > 0 {
		lastCreated = keys[0].CreatedAt
	}

	keyConfig, ok := service.keysConfig[request.Usage]
	if !ok {
		return nil, ErrConfigNotFound
	}

	span.SetAttributes(
		attribute.Int64("key.last_created", lastCreated.Unix()),
		attribute.Float64("key.rotation_interval", keyConfig.Key.Rotation.Seconds()),
	)

	var latestKey *dao.Jwk

	if time.Since(lastCreated) >= keyConfig.Key.Rotation {
		keyGenerator, ok := JwkGenerators[keyConfig.Alg]
		if !ok {
			return nil, otel.ReportError(span, fmt.Errorf("%w: %s", ErrJwkGenUnknownKeyUsage, request.Usage))
		}

		privateKey, publicKey, privateKID, publicKID, err := keyGenerator()
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("generate key: %w", err))
		}

		span.AddEvent("key.generated", trace.WithAttributes(
			attribute.String("key.private.kid", privateKID),
			attribute.String("key.public.kid", publicKID),
			attribute.String("key.alg", string(keyConfig.Alg)),
		))

		// Encrypt the private key using the master key, so it is protected against database dumping.
		privateKeyEncrypted, err := lib.EncryptMasterKey(ctx, privateKey)
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("encrypt private key: %w", err))
		}

		span.AddEvent("key.private.encrypted")

		privateKeyEncoded := base64.RawURLEncoding.EncodeToString(privateKeyEncrypted)

		span.AddEvent("key.private.encoded")

		// Both private and public keys share the same KID.
		kid, err := uuid.Parse(privateKID)
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("parse KID: %w", err))
		}

		var publicKeyEncoded *string

		if publicKey != nil {
			publicKeySerialized, err := json.Marshal(publicKey)
			if err != nil {
				return nil, otel.ReportError(span, fmt.Errorf("serialize public key: %w", err))
			}

			publicKeyEncoded = lo.ToPtr(base64.RawURLEncoding.EncodeToString(publicKeySerialized))

			span.AddEvent("key.public.encoded")
		}

		now := time.Now()

		latestKey, err = service.repositoryInsert.Exec(ctx, &dao.JwkInsertRequest{
			ID:         kid,
			PrivateKey: privateKeyEncoded,
			PublicKey:  publicKeyEncoded,
			Usage:      request.Usage,
			Now:        now,
			Expiration: now.Add(keyConfig.Key.TTL),
		})
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("insert key: %w", err))
		}

		span.AddEvent("key.inserted")
	} else {
		span.AddEvent("skipped")

		latestKey = keys[0]
	}

	output, err := service.serviceExtract.Exec(ctx, &JwkExtractRequest{
		Jwk:     latestKey,
		Private: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, output), nil
}
