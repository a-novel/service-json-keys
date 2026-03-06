package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

type JwkGetPublicService interface {
	Exec(ctx context.Context, request *services.JwkSelectRequest) (*services.Jwk, error)
}

type JwkGetPublic struct {
	service JwkGetPublicService
}

func NewJwkGetPublic(service JwkGetPublicService) *JwkGetPublic {
	return &JwkGetPublic{service: service}
}

func (handler *JwkGetPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.JwkGetPublic")
	defer span.End()

	keyId, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		_ = otel.ReportError(span, err)

		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

		return
	}

	jwk, err := handler.service.Exec(ctx, &services.JwkSelectRequest{ID: keyId})
	if errors.Is(err, dao.ErrJwkSelectNotFound) {
		_ = otel.ReportError(span, err)

		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)

		return
	}

	if err != nil {
		_ = otel.ReportError(span, err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	httpf.SendJSON(ctx, w, span, jwk)
}
