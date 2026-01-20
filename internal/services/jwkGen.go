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
	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

var ErrJwkGenUnknownKeyUsage = errors.New("unknown key request.Usage")

// KeyGenerator is a generic method that generates a private/public key pair.
type KeyGenerator func() (privateKey, publicKey *jwa.JWK, err error)

type JwkGenRepositorySearch interface {
	Exec(ctx context.Context, request *dao.JwkSearchRequest) ([]*dao.Jwk, error)
}

type JwkGenRepositoryInsert interface {
	Exec(ctx context.Context, request *dao.JwkInsertRequest) (*dao.Jwk, error)
}

type JwkGenServiceExtract interface {
	Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error)
}

type JwkGenRequest struct {
	// The intended usage of the token. It will be used to select
	// the relevant Json Web Key configuration.
	Usage string
}

// JwkGen is the service responsible for generating new keys.
//
// It does not force the generation of a key. Instead, when called, it checks the
// current state of the database to determine if a new main key should be generated
// or not.
//
// If the main key for the target usage is recent enough, this service will return it
// and skip generation. This will be indicated in traces.
type JwkGen struct {
	repositorySearch JwkGenRepositorySearch
	repositoryInsert JwkGenRepositoryInsert
	serviceExtract   JwkGenServiceExtract
	keysConfig       map[string]*config.Jwk
}

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
	ctx, span := otel.Tracer().Start(ctx, "service.JwkGen")
	defer span.End()

	span.SetAttributes(attribute.String("request.Usage", request.Usage))

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
		return nil, otel.ReportError(span, ErrConfigNotFound)
	}

	span.SetAttributes(
		attribute.Int64("lastCreated", lastCreated.Unix()),
		attribute.Float64("rotationInterval", keyConfig.Key.Rotation.Seconds()),
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

		span.AddEvent("keyGenerated", trace.WithAttributes(
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

		// Encode values to base64 before saving them.
		privateKeyEncoded := base64.RawURLEncoding.EncodeToString(privateKeyEncrypted)

		span.AddEvent("key.private.encoded")

		// Extract the KID from the private key. Both public and private key should share the same KID.
		kid, err := uuid.Parse(privateKID)
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("parse KID: %w", err))
		}

		var publicKeyEncoded *string

		if publicKey != nil {
			// Serialize the public key.
			publicKeySerialized, err := json.Marshal(publicKey)
			if err != nil {
				return nil, otel.ReportError(span, fmt.Errorf("serialize public key: %w", err))
			}

			publicKeyEncoded = lo.ToPtr(base64.RawURLEncoding.EncodeToString(publicKeySerialized))

			span.AddEvent("key.public.encoded")
		}

		// Insert the new key in the database.
		latestKey, err = service.repositoryInsert.Exec(ctx, &dao.JwkInsertRequest{
			ID:         kid,
			PrivateKey: privateKeyEncoded,
			PublicKey:  publicKeyEncoded,
			Usage:      request.Usage,
			Now:        time.Now(),
			Expiration: time.Now().Add(keyConfig.Key.TTL),
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
