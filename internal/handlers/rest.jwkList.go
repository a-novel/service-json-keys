package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// RestJwkListService is the service dependency of [RestJwkList].
type RestJwkListService interface {
	Exec(ctx context.Context, request *services.JwkSearchRequest) ([]*services.Jwk, error)
}

// RestJwkList is the REST handler that returns the active public keys for a given usage,
// reading the usage from the "usage" query parameter.
type RestJwkList struct {
	service RestJwkListService
	logger  logging.Log
}

// NewRestJwkList returns a new RestJwkList handler backed by the given service.
func NewRestJwkList(service RestJwkListService, logger logging.Log) *RestJwkList {
	return &RestJwkList{service: service, logger: logger}
}

func (handler *RestJwkList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.JwkList")
	defer span.End()

	usage := r.URL.Query().Get("usage")

	jwks, err := handler.service.Exec(ctx, &services.JwkSearchRequest{Usage: usage})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, jwks)
}
