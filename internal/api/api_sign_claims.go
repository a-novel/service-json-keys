package api

import (
	"context"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
)

type SignClaimsService interface {
	SignClaims(ctx context.Context, request services.SignClaimsRequest) (string, error)
}

func (api *API) SignClaims(
	ctx context.Context, req codegen.SignClaimsReq, params codegen.SignClaimsParams,
) (codegen.SignClaimsRes, error) {
	span := sentry.StartSpan(ctx, "API.SignClaims")
	defer span.Finish()

	span.SetData("usage", params.Usage)

	token, err := api.SignClaimsService.SignClaims(span.Context(), services.SignClaimsRequest{
		Claims: req,
		Usage:  models.KeyUsage(params.Usage),
	})
	if err != nil {
		return nil, err
	}

	span.SetData("token", token)

	return &codegen.Token{Token: token}, nil
}
