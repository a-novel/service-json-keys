package services

import (
	"context"

	"github.com/a-novel-kit/jwt/jwa"
)

// JwkExportLocal is a service used to wrap the local application so it can be used as a
// cached jwk.Source to reduce load on the database.
//
// This exporter is meant for internal usage only. The pkg defines its own exporter using the
// grpc api. See pkg.jwkExportGrpc.
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
