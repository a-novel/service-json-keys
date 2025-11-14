package services

import (
	"context"

	"github.com/a-novel-kit/jwt/jwa"
)

type JwkExportLocal struct {
	service *JwkSearch
}

func NewJwkExportLocal(service *JwkSearch) *JwkExportLocal {
	return &JwkExportLocal{service: service}
}

func (source *JwkExportLocal) SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error) {
	return source.service.Exec(ctx, &JwkSearchRequest{
		Usage:   usage,
		Private: true,
	})
}
