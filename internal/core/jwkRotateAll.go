package core

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

// JwkRotateAllServiceGen is the per-usage generation dependency of [JwkRotateAll].
type JwkRotateAllServiceGen interface {
	Exec(ctx context.Context, request *JwkGenRequest) (*Jwk, error)
}

// JwkRotateAllRequest holds the parameters for a [JwkRotateAll.Exec] call. It is
// empty because the set of usages to rotate is configuration, not a caller's
// choice — but it exists so the operation can gain one without breaking callers.
type JwkRotateAllRequest struct{}

// JwkRotateAllResponse reports the outcome of a [JwkRotateAll.Exec] call.
type JwkRotateAllResponse struct {
	// Processed counts the usages that were ensured. It is only meaningful when the
	// call returned no error: a partial count describes work that has been rolled
	// back, and reporting it would say keys were rotated when none were.
	Processed int
}

// A JwkRotateAll ensures every configured usage has a current key, as a single
// unit of work: a failure partway through leaves none of them rotated.
//
// That atomicity is the reason it takes a transactor rather than reaching for one.
// The rotation used to open a transaction and then hand each generation the
// surrounding context, so every key committed on its own and a failure on the
// third usage left the first two rotated — while the job reported failure as
// though nothing had been written.
type JwkRotateAll struct {
	serviceGen JwkRotateAllServiceGen
	transactor transaction.Transactor
	keysConfig map[string]*config.Jwk
}

// NewJwkRotateAll returns a JwkRotateAll rotating the usages declared in keysConfig.
func NewJwkRotateAll(
	serviceGen JwkRotateAllServiceGen,
	transactor transaction.Transactor,
	keysConfig map[string]*config.Jwk,
) *JwkRotateAll {
	return &JwkRotateAll{serviceGen: serviceGen, transactor: transactor, keysConfig: keysConfig}
}

func (service *JwkRotateAll) Exec(
	ctx context.Context, _ *JwkRotateAllRequest,
) (*JwkRotateAllResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.JwkRotateAll")
	defer span.End()

	span.SetAttributes(attribute.Int("keys.usages", len(service.keysConfig)))

	processed := 0

	err := service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		for usage := range service.keysConfig {
			_, err := service.serviceGen.Exec(ctx, &JwkGenRequest{Usage: usage})
			if err != nil {
				return fmt.Errorf("generate key for usage %s: %w", usage, err)
			}

			processed++
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("rotate keys: %w", err))
	}

	span.SetAttributes(attribute.Int("keys.processed", processed))

	return otel.ReportSuccess(span, &JwkRotateAllResponse{Processed: processed}), nil
}
