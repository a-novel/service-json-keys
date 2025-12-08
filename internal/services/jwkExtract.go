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

type JwkExtractRequest struct {
	// The dao object to extract from.
	Jwk *dao.Jwk
	// Whether to extract the private key or not. If false, only the public
	// key is returned.
	Private bool
}

// JwkExtract decodes the raw keys returned from the DAO layer.
type JwkExtract struct{}

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
		// In case of a symmetric jwk, the public member will be nil, and the private member will be returned
		// instead.
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
		// Private jwk also needs to be decrypted.
		func() error { return lib.DecryptMasterKey(ctx, decoded, &deserialized) },
		func() error { return json.Unmarshal(decoded, &deserialized) },
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("deserialize request.Jwk: %w", err))
	}

	return otel.ReportSuccess(span, deserialized), nil
}
