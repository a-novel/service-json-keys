package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/core"
)

// RestAPIKeyGetService is the service dependency of [RestAPIKeyGet].
type RestAPIKeyGetService interface {
	Exec(ctx context.Context, request *core.APIKeySelectRequest) (*core.APIKey, error)
}

// RestAPIKeyGet is the REST handler that returns an API key by its ID, reading the ID from the "id"
// query parameter. A cache revalidates the key on every use, so it learns a change of state at
// once; see [sendPublicKeys].
type RestAPIKeyGet struct {
	service RestAPIKeyGetService
	logger  logging.Log
}

func NewRestAPIKeyGet(service RestAPIKeyGetService, logger logging.Log) *RestAPIKeyGet {
	return &RestAPIKeyGet{service: service, logger: logger}
}

// restAPIKey is the JSON form of an API key.
type restAPIKey struct {
	ID          string    `json:"id"`
	Purpose     string    `json:"purpose"`
	Key         *core.Jwk `json:"key"`
	Deactivated bool      `json:"deactivated"`
	Leaked      bool      `json:"leaked"`
}

func (handler *RestAPIKeyGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.APIKeyGet")
	defer span.End()

	key, err := handler.service.Exec(ctx, &core.APIKeySelectRequest{ID: r.URL.Query().Get("id")})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{core.ErrAPIKeyNotFound: http.StatusNotFound}, err)

		return
	}

	err = sendPublicKeys(w, r, restAPIKey{
		ID:          key.ID,
		Purpose:     key.Purpose,
		Key:         key.Key,
		Deactivated: key.Deactivated,
		Leaked:      key.Leaked,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{}, err)
	}
}
