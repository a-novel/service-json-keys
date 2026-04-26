package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

const (
	// RestHealthStatusUp is the JSON status value reported when a dependency is healthy.
	RestHealthStatusUp = "up"
	// RestHealthStatusDown is the JSON status value reported when a dependency is unhealthy.
	RestHealthStatusDown = "down"
)

// RestHealthStatus is the JSON representation of a single dependency's health.
// The shape is deliberately minimal: /healthcheck is an unauthenticated public
// endpoint, so it must not expose raw error messages — those routinely include
// internal hostnames, ports, or schema names that leak infrastructure topology.
// The underlying error is recorded on the trace span for operators instead.
type RestHealthStatus struct {
	// Status is either [RestHealthStatusUp] or [RestHealthStatusDown].
	Status string `json:"status"`
}

// NewRestHealthStatus converts an error into a RestHealthStatus, mapping nil to
// [RestHealthStatusUp] and any non-nil error to [RestHealthStatusDown]. The error
// itself is intentionally discarded from the public response; see [RestHealthStatus].
func NewRestHealthStatus(err error) *RestHealthStatus {
	return &RestHealthStatus{
		Status: lo.Ternary(err == nil, RestHealthStatusUp, RestHealthStatusDown),
	}
}

// RestHealth is the HTTP handler that reports the operational health of the service
// and its dependencies as a JSON object.
type RestHealth struct{}

// NewRestHealth returns a new RestHealth handler.
func NewRestHealth() *RestHealth {
	return &RestHealth{}
}

func (handler *RestHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Health")
	defer span.End()

	httpf.SendJSON(ctx, w, span, map[string]any{
		"client:postgres": NewRestHealthStatus(handler.reportPostgres(ctx)),
	})
}

func (handler *RestHealth) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportPostgres)")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		// Cannot assess the DB connection in transaction mode.
		return nil
	}

	err = pgdb.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
