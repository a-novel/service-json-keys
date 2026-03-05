package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/services"
)

type JwkListPublicService interface {
	Exec(ctx context.Context, request *services.JwkSearchRequest) ([]*services.Jwk, error)
}

type JwkListPublic struct {
	service JwkListPublicService
}

func NewJwkListPublic(service JwkListPublicService) *JwkListPublic {
	return &JwkListPublic{service: service}
}

func (handler *JwkListPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.JwkListPublic")
	defer span.End()

	usage := r.URL.Query().Get("usage")

	jwks, err := handler.service.Exec(ctx, &services.JwkSearchRequest{Usage: usage})
	if err != nil {
		_ = otel.ReportError(span, err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	httpf.SendJSON(ctx, w, span, jwks)
}
