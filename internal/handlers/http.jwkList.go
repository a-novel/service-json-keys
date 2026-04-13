package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// JwkListPublicService is the service dependency of [JwkListPublic].
type JwkListPublicService interface {
	Exec(ctx context.Context, request *services.JwkSearchRequest) ([]*services.Jwk, error)
}

// JwkListPublic is the HTTP handler that returns the active public keys for a given usage,
// reading the usage from the "usage" query parameter.
type JwkListPublic struct {
	service JwkListPublicService
	logger  logging.Log
}

// NewJwkListPublic returns a new JwkListPublic handler backed by the given service.
func NewJwkListPublic(service JwkListPublicService, logger logging.Log) *JwkListPublic {
	return &JwkListPublic{service: service, logger: logger}
}

func (handler *JwkListPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.JwkListPublic")
	defer span.End()

	usage := r.URL.Query().Get("usage")

	jwks, err := handler.service.Exec(ctx, &services.JwkSearchRequest{Usage: usage})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, jwks)
}
