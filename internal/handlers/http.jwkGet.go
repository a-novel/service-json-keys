package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// JwkGetPublicService is the service dependency of [JwkGetPublic].
type JwkGetPublicService interface {
	Exec(ctx context.Context, request *services.JwkSelectRequest) (*services.Jwk, error)
}

// JwkGetPublic is the HTTP handler that returns a single public JWK by its ID,
// reading the key ID from the "id" query parameter.
type JwkGetPublic struct {
	service JwkGetPublicService
	logger  logging.Log
}

// NewJwkGetPublic returns a new JwkGetPublic handler backed by the given service.
func NewJwkGetPublic(service JwkGetPublicService, logger logging.Log) *JwkGetPublic {
	return &JwkGetPublic{service: service, logger: logger}
}

func (handler *JwkGetPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.JwkGetPublic")
	defer span.End()

	rawID := r.URL.Query().Get("id")

	keyID, err := uuid.Parse(rawID)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	jwk, err := handler.service.Exec(ctx, &services.JwkSelectRequest{ID: keyID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrJwkSelectNotFound: http.StatusNotFound,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, jwk)
}
