package handlers

import (
	"net/http"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel-kit/golib/otel"
)

// Ping is the HTTP handler that responds with "pong" for liveness checks.
type Ping struct{}

// NewPing returns a new Ping handler.
func NewPing() *Ping {
	return new(Ping)
}

func (handler *Ping) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer().Start(r.Context(), "rest.Ping")
	defer span.End()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))

	span.RecordError(err)
	span.SetStatus(lo.Ternary(err == nil, codes.Ok, codes.Error), "")
}
