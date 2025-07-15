package api

import (
	"context"
	"fmt"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/api"
)

type SignClaimsService interface {
	SignClaims(ctx context.Context, request services.SignClaimsRequest) (string, error)
}

func (api *API) SignClaims(
	ctx context.Context, req apimodels.SignClaimsReq, params apimodels.SignClaimsParams,
) (apimodels.SignClaimsRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.SignClaims")
	defer span.End()

	token, err := api.SignClaimsService.SignClaims(ctx, services.SignClaimsRequest{
		Claims: req,
		Usage:  models.KeyUsage(params.Usage),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("sign claims: %w", err))
	}

	return otel.ReportSuccess(span, &apimodels.Token{Token: token}), nil
}
