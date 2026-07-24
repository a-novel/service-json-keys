package core

import (
	"context"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2/jwa"
)

// JwkExportLocalSource is the service dependency of [JwkExportLocal] for fetching private keys.
type JwkExportLocalSource interface {
	Exec(ctx context.Context, request *JwkSearchRequest) ([]*Jwk, error)
}

// A JwkExportLocal wraps the local JwkSearch service as a jwk.Source, so private keys can
// be cached in memory and served without hitting the database on every request.
//
// Server-side only; the public client package has its own gRPC-backed exporter.
type JwkExportLocal struct {
	service JwkExportLocalSource
}

// NewJwkExportLocal returns a new JwkExportLocal service backed by the given search service.
func NewJwkExportLocal(service JwkExportLocalSource) *JwkExportLocal {
	return &JwkExportLocal{service: service}
}

func (source *JwkExportLocal) SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.JwkExportLocal.SearchKeys")
	defer span.End()

	return source.service.Exec(ctx, &JwkSearchRequest{
		Usage:   usage,
		Private: true,
	})
}
