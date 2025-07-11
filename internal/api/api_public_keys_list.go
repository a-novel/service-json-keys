package api

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
)

type SearchKeysService interface {
	SearchKeys(ctx context.Context, request services.SearchKeysRequest) ([]*jwa.JWK, error)
}

func (api *API) ListPublicKeys(
	ctx context.Context, params codegen.ListPublicKeysParams,
) (codegen.ListPublicKeysRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.ListPublicKeys")
	defer span.End()

	keys, err := api.SearchKeysService.SearchKeys(ctx, services.SearchKeysRequest{
		Usage:   models.KeyUsage(params.Usage),
		Private: false,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("search keys: %w", err))
	}

	keysModels, err := api.jwksToModels(keys...)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("convert keys to models: %w", err))
	}

	return otel.ReportSuccess(span, lo.ToPtr(codegen.ListPublicKeysOKApplicationJSON(keysModels))), nil
}
