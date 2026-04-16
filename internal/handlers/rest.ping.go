package handlers

import (
	"net/http"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel-kit/golib/otel"
)

// RestPing is the REST handler that responds with "pong" for liveness checks.
type RestPing struct{}

// NewRestPing returns a new RestPing handler.
func NewRestPing() *RestPing {
	return new(RestPing)
}

func (handler *RestPing) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer().Start(r.Context(), "rest.Ping")
	defer span.End()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))

	span.RecordError(err)
	span.SetStatus(lo.Ternary(err == nil, codes.Ok, codes.Error), "")
}
