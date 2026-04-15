package services

import (
	"context"

	"github.com/a-novel-kit/jwt/jwa"
)

// JwkExportLocalSource is the service dependency of [JwkExportLocal] for fetching private keys.
type JwkExportLocalSource interface {
	Exec(ctx context.Context, request *JwkSearchRequest) ([]*Jwk, error)
}

// A JwkExportLocal wraps the local JwkSearch service as a jwk.Source, so private keys can
// be cached in memory and served without hitting the database on every request.
//
// This exporter is for internal (server-side) use only. The public client package uses its
// own gRPC-backed exporter instead.
type JwkExportLocal struct {
	service JwkExportLocalSource
}

// NewJwkExportLocal returns a new JwkExportLocal service backed by the given search service.
func NewJwkExportLocal(service JwkExportLocalSource) *JwkExportLocal {
	return &JwkExportLocal{service: service}
}

func (source *JwkExportLocal) SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error) {
	return source.service.Exec(ctx, &JwkSearchRequest{
		Usage:   usage,
		Private: true,
	})
}
