package pkg

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/internal/adapters"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/api"
)

type ClaimsVerifier[Out any] struct {
	service *services.VerifyClaimsService[Out]
}

func NewClaimsVerifier[Out any](
	client *apimodels.Client,
	keys map[models.KeyUsage]*models.JSONKeyConfig,
) (*ClaimsVerifier[Out], error) {
	adapter := adapters.NewPublicKeySourcesAPI(client)

	source, err := services.NewPublicKeySource(adapter, keys)
	if err != nil {
		return nil, fmt.Errorf("NewPublicKeySourcesAPI: %w", err)
	}

	recipients, err := services.NewRecipients(source, keys)
	if err != nil {
		return nil, fmt.Errorf("NewRecipients: %w", err)
	}

	service := services.NewVerifyClaimsService[Out](recipients, keys)

	return &ClaimsVerifier[Out]{service: service}, nil
}

type VerifyClaimsOptions struct {
	IgnoreExpired bool
}

func (pkg *ClaimsVerifier[Out]) VerifyClaims(
	ctx context.Context, usage models.KeyUsage, accessToken string, options *VerifyClaimsOptions,
) (*Out, error) {
	return pkg.service.VerifyClaims(ctx, services.VerifyClaimsRequest{
		Token:         accessToken,
		Usage:         usage,
		IgnoreExpired: lo.FromPtr(options).IgnoreExpired,
	})
}
