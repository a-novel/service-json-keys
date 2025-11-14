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

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

var ErrJwkGenUnknownKeyUsage = errors.New("unknown key request.Usage")

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
	Usage string
}

type JwkGen struct {
	repositorySearch JwkGenRepositorySearch
	repositoryInsert JwkGenRepositoryInsert
	serviceExtract   JwkGenServiceExtract
	keys             map[string]*config.Jwk
}

func NewJwkGen(
	repositorySearch JwkGenRepositorySearch,
	repositoryInsert JwkGenRepositoryInsert,
	serviceExtract JwkGenServiceExtract,
	keys map[string]*config.Jwk,
) *JwkGen {
	return &JwkGen{
		repositorySearch: repositorySearch,
		repositoryInsert: repositoryInsert,
		serviceExtract:   serviceExtract,
		keys:             keys,
	}
}

func (service *JwkGen) Exec(ctx context.Context, request *JwkGenRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JwkGen")
	defer span.End()

	span.SetAttributes(attribute.String("request.Usage", request.Usage))

	// Check the time last key was inserted for this request.Usage, and compare to config. If last key is too recent,
	// return without generating a new key.
	keys, err := service.repositorySearch.Exec(ctx, &dao.JwkSearchRequest{Usage: request.Usage})
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
		attribute.Float64("rotationInterval", service.keys[request.Usage].Key.Rotation.Seconds()),
	)

	var latestKey *dao.Jwk

	if time.Since(lastCreated) >= service.keys[request.Usage].Key.Rotation {
		keyGenerator, ok := JwkGenerators[service.keys[request.Usage].Alg]
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
			attribute.String("key.alg", string(service.keys[request.Usage].Alg)),
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
			Expiration: time.Now().Add(service.keys[request.Usage].Key.TTL),
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
