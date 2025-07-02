package adapters

import (
	"context"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
)

type PrivateKeySourcesLocal struct {
	service *services.SearchKeysService
}

func NewPrivateKeySourcesLocalAdapter(service *services.SearchKeysService) *PrivateKeySourcesLocal {
	return &PrivateKeySourcesLocal{service: service}
}

func (source *PrivateKeySourcesLocal) SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*jwa.JWK, error) {
	return source.service.SearchKeys(ctx, services.SearchKeysRequest{
		Usage:   usage,
		Private: true,
	})
}
