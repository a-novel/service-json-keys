package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

// JwkExtractRequest holds the parameters for a [JwkExtract.Exec] call.
type JwkExtractRequest struct {
	// Jwk is the DAO entity to extract key material from.
	Jwk *dao.Jwk
	// Private indicates whether to return the private key; when false, the public key is returned.
	Private bool
}

// A JwkExtract decodes the raw keys returned from the DAO layer.
type JwkExtract struct{}

// NewJwkExtract returns a new JwkExtract service.
func NewJwkExtract() *JwkExtract {
	return new(JwkExtract)
}

func (service *JwkExtract) Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.JwkExtract")
	defer span.End()

	span.SetAttributes(
		attribute.Bool("request.Jwk.request.Private", request.Private),
		attribute.String("request.Jwk.ID", request.Jwk.ID.String()),
		attribute.String("request.Jwk.Usage", request.Jwk.Usage),
		attribute.Int64("request.Jwk.CreatedAt", request.Jwk.CreatedAt.Unix()),
		attribute.Int64("request.Jwk.ExpiresAt", request.Jwk.ExpiresAt.Unix()),
	)

	decoded, err := base64.RawURLEncoding.DecodeString(
		// For symmetric JWKs the public key is nil, so always fall back to the private key.
		lo.Ternary(
			request.Private || request.Jwk.PublicKey == nil,
			request.Jwk.PrivateKey,
			lo.FromPtr(request.Jwk.PublicKey),
		),
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("decode request.Jwk: %w", err))
	}

	var deserialized *Jwk

	err = lo.TernaryF(
		request.Private || request.Jwk.PublicKey == nil,
		// Private key material is encrypted at rest and must be decrypted before use.
		func() error { return lib.DecryptMasterKey(ctx, decoded, &deserialized) },
		func() error { return json.Unmarshal(decoded, &deserialized) },
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("deserialize request.Jwk: %w", err))
	}

	return otel.ReportSuccess(span, deserialized), nil
}
