package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// RestJwkGetService is the service dependency of [RestJwkGet].
type RestJwkGetService interface {
	Exec(ctx context.Context, request *services.JwkSelectRequest) (*services.Jwk, error)
}

// RestJwkGet is the REST handler that returns a single public JWK by its ID,
// reading the key ID from the "id" query parameter.
type RestJwkGet struct {
	service RestJwkGetService
	logger  logging.Log
}

// NewRestJwkGet returns a new RestJwkGet handler backed by the given service.
func NewRestJwkGet(service RestJwkGetService, logger logging.Log) *RestJwkGet {
	return &RestJwkGet{service: service, logger: logger}
}

func (handler *RestJwkGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.JwkGet")
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
			services.ErrJwkNotFound: http.StatusNotFound,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, jwk)
}
