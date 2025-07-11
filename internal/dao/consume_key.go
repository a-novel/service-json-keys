package dao

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/lib"
)

// ConsumeKey converts a key from DAO entity to aJWK object.
func ConsumeKey(ctx context.Context, key *KeyEntity, private bool) (*jwa.JWK, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ConsumeKey")
	defer span.End()

	span.SetAttributes(
		attribute.Bool("key.private", private),
		attribute.String("key.id", key.ID.String()),
		attribute.String("key.usage", key.Usage.String()),
		attribute.Int64("key.created_at", key.CreatedAt.Unix()),
		attribute.Int64("key.expires_at", key.ExpiresAt.Unix()),
	)

	decoded, err := base64.RawURLEncoding.DecodeString(
		// In case of a symmetric key, the public member will be nil, and the private member will be returned
		// instead.
		lo.Ternary(private || key.PublicKey == nil, key.PrivateKey, lo.FromPtr(key.PublicKey)),
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("decode key: %w", err))
	}

	var deserialized *jwa.JWK

	err = lo.TernaryF(
		private || key.PublicKey == nil,
		// Private keys also needs to be decrypted.
		func() error { return lib.DecryptMasterKey(ctx, decoded, &deserialized) },
		func() error { return json.Unmarshal(decoded, &deserialized) },
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("deserialize key: %w", err))
	}

	return otel.ReportSuccess(span, deserialized), nil
}
