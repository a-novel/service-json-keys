package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/services"
)

type JwkListPublicService interface {
	Exec(ctx context.Context, request *services.JwkSearchRequest) ([]*services.Jwk, error)
}

type JwkListPublic struct {
	service JwkListPublicService
	logger  logging.Log
}

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
