package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/core"
)

// RestJwkListService is the service dependency of [RestJwkList].
type RestJwkListService interface {
	Exec(ctx context.Context, request *core.JwkSearchRequest) ([]*core.Jwk, error)
}

// RestJwkList is the REST handler that returns the active public keys for a given usage,
// reading the usage from the "usage" query parameter.
type RestJwkList struct {
	service RestJwkListService
	logger  logging.Log
}

func NewRestJwkList(service RestJwkListService, logger logging.Log) *RestJwkList {
	return &RestJwkList{service: service, logger: logger}
}

func (handler *RestJwkList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.JwkList")
	defer span.End()

	jwks, err := handler.service.Exec(ctx, &core.JwkSearchRequest{Usage: r.URL.Query().Get("usage")})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{}, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, jwks)
}
