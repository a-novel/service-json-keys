package handlers

import (
	"context"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
)

func NewHealthStatus(err error) *protogen.DependencyHealth {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return &protogen.DependencyHealth{
		Status: lo.Ternary(
			err == nil,
			protogen.DependencyStatus_DEPENDENCY_STATUS_UP,
			protogen.DependencyStatus_DEPENDENCY_STATUS_DOWN,
		),
		Err: errMsg,
	}
}

type Status struct {
	protogen.UnimplementedStatusServiceServer
}

func NewStatus() *Status {
	return new(Status)
}

func (handler *Status) Status(ctx context.Context, _ *protogen.StatusRequest) (*protogen.StatusResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.Status")
	defer span.End()

	return &protogen.StatusResponse{
		Postgres: NewHealthStatus(handler.reportPostgres(ctx)),
	}, nil
}

func (handler *Status) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "api.Status(reportPostgres)")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		// Cannot assess db connection if we are running on transaction mode
		return nil
	}

	err = pgdb.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
