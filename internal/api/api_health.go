package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/models/api"
)

func (api *API) Ping(_ context.Context) (apimodels.PingRes, error) {
	return &apimodels.PingOK{Data: strings.NewReader("pong")}, nil
}

func (api *API) reportPostgres(ctx context.Context) apimodels.Dependency {
	ctx, span := otel.Tracer().Start(ctx, "api.reportPostgres")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		_ = otel.ReportError(span, err)

		return apimodels.Dependency{
			Name:   "postgres",
			Status: apimodels.DependencyStatusDown,
		}
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		_ = otel.ReportError(span, fmt.Errorf("retrieve postgres context: invalid type %T", pg))

		return apimodels.Dependency{
			Name:   "postgres",
			Status: apimodels.DependencyStatusDown,
		}
	}

	err = pgdb.Ping()
	if err != nil {
		_ = otel.ReportError(span, err)

		return apimodels.Dependency{
			Name:   "postgres",
			Status: apimodels.DependencyStatusDown,
		}
	}

	span.SetStatus(codes.Ok, "")

	return apimodels.Dependency{
		Name:   "postgres",
		Status: apimodels.DependencyStatusUp,
	}
}

func (api *API) Healthcheck(ctx context.Context) (apimodels.HealthcheckRes, error) {
	return &apimodels.Health{
		Postgres: api.reportPostgres(ctx),
	}, nil
}
