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

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/config"
)

var ErrUnknownKeyUsage = errors.New("unknown key usage")

// KeyGenerator generates a new JSON Web Key private/public pair. It is a key-type agnostic wrapper around the
// JWT library generators.
type KeyGenerator func() (privateKey, publicKey *jwa.JWK, err error)

type GenerateKeySource interface {
	SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*dao.KeyEntity, error)
	InsertKey(ctx context.Context, data dao.InsertKeyData) (*dao.KeyEntity, error)
}

func NewGenerateKeySource(searchDAO *dao.SearchKeysRepository, insertDAO *dao.InsertKeyRepository) GenerateKeySource {
	return &struct {
		dao.SearchKeysRepository
		dao.InsertKeyRepository
	}{
		SearchKeysRepository: *searchDAO,
		InsertKeyRepository:  *insertDAO,
	}
}

type GenerateKeyService struct {
	source GenerateKeySource
	keys   map[models.KeyUsage]*config.JWKS
}

func NewGenerateKeyService(
	source GenerateKeySource,
	keys map[models.KeyUsage]*config.JWKS,
) *GenerateKeyService {
	return &GenerateKeyService{source: source, keys: keys}
}

// GenerateKey generates a new key pair for a given usage. It uses the generateKeysConfig to generate the
// correct payload. Private key is encrypted using the master key before being saved in the database.
func (service *GenerateKeyService) GenerateKey(ctx context.Context, usage models.KeyUsage) (*uuid.UUID, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.GenerateKey")
	defer span.End()

	span.SetAttributes(attribute.String("usage", usage.String()))

	// Check the time last key was inserted for this usage, and compare to config. If last key is too recent,
	// return without generating a new key.
	keys, err := service.source.SearchKeys(ctx, usage)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list keys: %w", err))
	}

	span.AddEvent("keys.retrieved", trace.WithAttributes(attribute.Int("keys.count", len(keys))))

	var lastCreated time.Time
	if len(keys) > 0 {
		lastCreated = keys[0].CreatedAt
	}

	span.SetAttributes(
		attribute.Int64("lastCreated", lastCreated.Unix()),
		attribute.Float64("rotationInterval", service.keys[usage].Key.Rotation.Seconds()),
	)

	// Last key was created within the rotation interval. No need to generate a new key.
	if time.Since(lastCreated) < service.keys[usage].Key.Rotation {
		span.AddEvent("skipped")

		return otel.ReportSuccess(span, &keys[0].ID), nil
	}

	keyGenerator, ok := KeyGenerators[service.keys[usage].Alg]
	if !ok {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", ErrUnknownKeyUsage, usage))
	}

	privateKey, publicKey, privateKID, publicKID, err := keyGenerator()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate key: %w", err))
	}

	span.AddEvent("keyGenerated", trace.WithAttributes(
		attribute.String("key.private.kid", privateKID),
		attribute.String("key.public.kid", publicKID),
		attribute.String("key.alg", string(service.keys[usage].Alg)),
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
	insertData := dao.InsertKeyData{
		ID:         kid,
		PrivateKey: privateKeyEncoded,
		PublicKey:  publicKeyEncoded,
		Usage:      usage,
		Now:        time.Now(),
		Expiration: time.Now().Add(service.keys[usage].Key.TTL),
	}

	_, err = service.source.InsertKey(ctx, insertData)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("insert key: %w", err))
	}

	span.AddEvent("key.inserted")

	return otel.ReportSuccess(span, &kid), nil
}
