package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

// ErrJwkExtractNoPublicKey is returned when a caller asks for the public half of a
// key that has none. Symmetric keys are stored without one, and are served as
// private material or not at all.
var ErrJwkExtractNoPublicKey = errors.New("jwk has no public key")

// JwkExtractRequest holds the parameters for a [JwkExtract.Exec] call.
type JwkExtractRequest struct {
	// Jwk is the DAO entity to extract key material from.
	Jwk *dao.Jwk
	// Private authorizes private key material. Without it, a key that has only private
	// material yields [ErrJwkExtractNoPublicKey].
	//
	// The signing path and the rotation job set it, and the public REST API never does. The gate
	// lives here because every caller passes through it.
	Private bool
}

// A JwkExtract decodes the raw keys returned from the DAO layer.
type JwkExtract struct{}

func NewJwkExtract() *JwkExtract {
	return &JwkExtract{}
}

func (service *JwkExtract) Exec(ctx context.Context, request *JwkExtractRequest) (*Jwk, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.JwkExtract")
	defer span.End()

	span.SetAttributes(
		attribute.Bool("key.private", request.Private),
		attribute.String("key.id", request.Jwk.ID.String()),
		attribute.String("key.usage", request.Jwk.Usage),
		attribute.Int64("key.created_at", request.Jwk.CreatedAt.Unix()),
		attribute.Int64("key.expires_at", request.Jwk.ExpiresAt.Unix()),
	)

	// A symmetric key has no public half, so a caller without authorization for
	// private material has nothing here it may be served.
	if !request.Private && request.Jwk.PublicKey == nil {
		return nil, otel.ReportError(span, ErrJwkExtractNoPublicKey)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(
		lo.Ternary(request.Private, request.Jwk.PrivateKey, lo.FromPtr(request.Jwk.PublicKey)),
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("decode request.Jwk: %w", err))
	}

	var deserialized *Jwk

	if request.Private {
		// Private key material is encrypted at rest.
		err = lib.DecryptMasterKey(ctx, decoded, &deserialized)
	} else {
		err = json.Unmarshal(decoded, &deserialized)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("deserialize request.Jwk: %w", err))
	}

	return deserialized, nil
}
